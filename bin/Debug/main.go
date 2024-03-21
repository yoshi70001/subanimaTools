package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

func getDriveService() (*drive.Service, error) {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		fmt.Printf("Unable to read credentials.json file. Err: %v\n", err)
		return nil, err
	}

	// If you want to modifyt this scope, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)

	if err != nil {
		return nil, err
	}

	client := getClient(config)

	service, err := drive.New(client)

	if err != nil {
		fmt.Printf("Cannot create the Google Drive service: %v\n", err)
		return nil, err
	}

	return service, err
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	fmt.Println("Paste Authrization code here :")
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// func createFolder(service *drive.Service, name string, parentId string) (*drive.File, error) {
// 	d := &drive.File{
// 		Name:     name,
// 		MimeType: "application/vnd.google-apps.folder",
// 		Parents:  []string{parentId},
// 	}

// 	file, err := service.Files.Create(d).Do()

// 	if err != nil {
// 		log.Println("Could not create dir: " + err.Error())
// 		return nil, err
// 	}

// 	return file, nil
// }

func createFile(service *drive.Service, name string, mimeType string, content io.Reader, parentId string) (*drive.File, error) {
	f := &drive.File{
		MimeType: mimeType,
		Name:     name,
		Parents:  []string{parentId},
	}
	file, err := service.Files.Create(f).Media(content).Do()

	if err != nil {
		time.Sleep(2 * time.Second)
		return createFile(service, name, "application/vnd.google-apps.document", content, parentId)
		// log.Println("Could not create file: " + err.Error())
		// return nil, err
	}

	return file, nil
}
func uploadFile(e os.DirEntry, srv *drive.Service, folderId string, wg *sync.WaitGroup, textasbyte *[]byte) {
	// fmt.Println(e.Name())
	f, err := os.Open("./AUTnFLostUniverse25FAB93F19ByJulian12100/" + e.Name())

	if err != nil {
		panic(fmt.Sprintf("cannot open file: %v", err))
	}

	defer f.Close()

	file, err := createFile(srv, e.Name(), "application/vnd.google-apps.document", f, folderId)

	if err != nil {
		panic(fmt.Sprintf("Could not create file: %v\n", err))
	}

	// fmt.Printf("File '%s' uploaded successfully", file.Name)
	// fmt.Printf("\nFile Id: '%s' ", file.Id)

	res, err := srv.Files.Export(
		file.Id,
		"text/plain", // mimeType is this.
	).Download()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	err2 := srv.Files.Delete(file.Id).Do()
	if err2 != nil {
		log.Fatalf("Error: %v", err2)
	}
	result, _ := ioutil.ReadAll(res.Body)
	lineSubtitle := strings.Split(string(result), "\n")
	lineSubtitle = slices.Delete(lineSubtitle, 0, 2)
	lines := strings.Join(lineSubtitle, "\n")
	fmt.Println(lines)
	*textasbyte = append(*textasbyte, []byte(lines)...)
	defer wg.Done()
}
func main() {
	var wg sync.WaitGroup
	var textasbyte []byte
	argsWithoutProg := os.Args[1:]
	directorio := "./subtitles" // Ruta del directorio que quieres verificar o crear

	if _, err := os.Stat(directorio); os.IsNotExist(err) {

		err := os.Mkdir(directorio, 0755) // 0755 es el permiso para el directorio (puedes cambiarlo según tus necesidades)
		if err != nil {
			fmt.Println("Error al crear el directorio:", err)
			return
		}
		fmt.Println("Directorio creado correctamente.")
	}

	srv, err := getDriveService()

	folderId := "root"

	entries, err := os.ReadDir(argsWithoutProg[0])
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		wg.Add(1)
		go uploadFile(e, srv, folderId, &wg, &textasbyte)
		time.Sleep(1 * time.Second)
	}
	wg.Wait() // Esperar a que todas las goroutines terminen

	fmt.Println("Todas las goroutines han finalizado.")
	os.WriteFile(directorio+"/"+argsWithoutProg[0]+".srt", textasbyte, 0666)

}
