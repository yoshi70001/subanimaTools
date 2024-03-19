#include <iostream>
#include <opencv2\highgui\highgui.hpp>
#include <opencv2\opencv.hpp>
#include <chrono>
#include <iomanip>
#include <filesystem>
#include <cstring>
#include <dirent.h>
using namespace std;
using namespace cv;

string times(int contador, double fps)
{
    int intPart = contador / fps;
    double fractPart = (contador / fps - intPart) * 1000;

    std::chrono::seconds sec(intPart);
    std::chrono::milliseconds ms(static_cast<int>(fractPart));
    auto h = std::chrono::duration_cast<std::chrono::hours>(sec);
    sec -= h;
    auto m = std::chrono::duration_cast<std::chrono::minutes>(sec);
    sec -= m;
    auto s = std::chrono::duration_cast<std::chrono::seconds>(sec);
    std::ostringstream oss;
    oss << std::setw(2) << std::setfill('0') << h.count() << "_"
        << std::setw(2) << std::setfill('0') << m.count() << "_"
        << std::setw(2) << std::setfill('0') << s.count() << "_"
        << std::setw(3) << std::setfill('0') << ms.count();
    return oss.str();
}

int textDetector(string rutavideo)
{

    VideoCapture cap(rutavideo);

    if (!cap.isOpened())
    {
        cout << "Error initializing video camera!" << endl;
        return -1;
    }
    double videoFPS = cap.get(cv::CAP_PROP_FPS);
    cv::dnn::Net model = cv::dnn::readNet("models/9-5-23.onnx");
    cv::dnn::Net modelTwo = cv::dnn::readNet("models/end2end.onnx");

    char* windowName = "mask";
    namedWindow(windowName, WINDOW_AUTOSIZE);
    char* windowName2 = "original";
    namedWindow(windowName2, WINDOW_AUTOSIZE);
    int frameCounter = 0;
    while (1)
    {

        Mat frame;

        bool bSuccess = cap.read(frame);

        if (!bSuccess)
        {
            cout << "Error reading frame from camera feed" << endl;
            break;
        }
        if(frameCounter%6==0)
        {
            Mat frameCropped;
            frame(Rect(0, frame.rows-(frame.rows/3), frame.cols, frame.rows/3)).copyTo(frameCropped);
            resize(frameCropped,frameCropped,cv::Size(256, 64));
            cv::Mat blob;
            cv::dnn::blobFromImage(frameCropped, blob, 1, cv::Size(256, 64));

            blob = blob.reshape(1, {1,64,256,3});
            model.setInput(blob);
            vector<double> percent = model.forward();

            if(percent[0]> 0.96)
            {
                Mat frameCropped2;
                Mat blob2;
                frame(Rect(0, frame.rows-(frame.rows/3), frame.cols, frame.rows/3)).copyTo(frameCropped2);
                cv::dnn::blobFromImage(frameCropped2/255, blob2, 1, cv::Size(frame.cols, frame.rows/3));
                blob2 = blob2.reshape(1, {1,3,frame.rows/3,frame.cols});
                modelTwo.setInput(blob2);
                Mat prob = modelTwo.forward().reshape(1, {frame.rows/3,frame.cols})*255;
                Mat output;
                prob.convertTo(output,CV_8U);
                cv::bitwise_not(output,output);
                cv::threshold(output,output,200,255,cv::THRESH_BINARY);
                cv::morphologyEx(output,output,cv::MORPH_OPEN,cv::Mat::ones(3, 3, CV_64F));
                cv::erode(output,output,cv::Mat::ones(3, 7, CV_64F),Point(-1,-1),2);
                imshow(windowName,output);

            };
            imshow(windowName2, frameCropped);
            switch (waitKey(1))
            {
            case 27:
                return 0;
            }
        }

        frameCounter+=1;
    }
    return 0;
}

int main()
{
    string directory_path = "videos";
    DIR* dir = opendir(directory_path.c_str());

    if (dir)
    {
        struct dirent* entry;
        while ((entry = readdir(dir)) != nullptr)
        {

            if(entry->d_name[0] == '.')
            {
                continue;
            };
            textDetector(directory_path+"/"+(entry->d_name));
        }
        closedir(dir);
    }
    else
    {
        cerr << "No se pudo abrir el directorio" << endl;
    }



    return 0;
}
