package main

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "net/url"
  "os"
  "os/user"
  "path/filepath"
  "sort"
  "strings"

  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "google.golang.org/api/drive/v3"

  "private/data"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
  cacheFile, err := tokenCacheFile()
  if err != nil {
    log.Fatalf("Unable to get path to cached credential file. %v", err)
  }
  tok, err := tokenFromFile(cacheFile)
  if err != nil {
    tok = getTokenFromWeb(config)
    saveToken(cacheFile, tok)
  }
  return config.Client(ctx, tok)
}

// GetTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
  authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
  fmt.Printf("Go to the following link in your browser then type the "+
    "authorization code: \n%v\n", authURL)

  var code string
  if _, err := fmt.Scan(&code); err != nil {
    log.Fatalf("Unable to read authorization code %v", err)
  }

  tok, err := config.Exchange(oauth2.NoContext, code)
  if err != nil {
    log.Fatalf("Unable to retrieve token from web %v", err)
  }
  return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
  usr, err := user.Current()
  if err != nil {
    return "", err
  }
  tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
  os.MkdirAll(tokenCacheDir, 0700)
  return filepath.Join(tokenCacheDir,
    url.QueryEscape("drive-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
  f, err := os.Open(file)
  if err != nil {
    return nil, err
  }
  t := &oauth2.Token{}
  err = json.NewDecoder(f).Decode(t)
  defer f.Close()
  return t, err
}

// saveToken uses a file path to reate a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
  fmt.Printf("Saving credential file to: %s\n", file)
  f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
  if err != nil {
    log.Fatalf("Unable to cache oauth token: %v", err)
  }
  defer f.Close()
  json.NewEncoder(f).Encode(token)
}

func main() {
  ctx := context.Background()

  b, err := ioutil.ReadFile("private/client_secret.json")
  if err != nil {
    log.Fatalf("Unable to read client secret file: %v", err)
  }

  // If modifying these scopes, delete your previously saved credentials
  // at ~/.credentials/drive-go-quickstart.json
  config, err := google.ConfigFromJSON(b, drive.DriveScope)
  if err != nil {
    log.Fatalf("Unable to parse client secret file to config: %v", err)
  }
  client := getClient(ctx, config)

  srv, err := drive.New(client)
  if err != nil {
    log.Fatalf("Unable to retrieve drive Client %v", err)
  }

  r, err := srv.Files.List().PageSize(1000).
    Fields("nextPageToken, files[name=\"Hello World Doc\" and not(trashed)](id, name, trashed)").
    OrderBy("name").Do()
  if err != nil {
    log.Fatalf("Unable to retrieve files: %v", err)
  }

  fmt.Println("Files:")
  if len(r.Files) > 0 {
    sort.Slice(r.Files, func(i, j int) bool { return r.Files[i].Name < r.Files[j].Name })
    for _, i := range r.Files {
      fmt.Printf("%s (%s %v)\n", i.Name, i.Id, i.Trashed)
    }
    fmt.Println("-------")
    fmt.Println(len(r.Files), "total files")
  } else {
    fmt.Println("No files found.")
    os.Exit(0)
  }

  // fmt.Println("Download a file (by id):")
  // var id string
  // if _, err := fmt.Scan(&id); err != nil {
  //   log.Fatalf("Unable to read file id to download: %v", err)
  // }
  // res, err := srv.Files.Get(id).Download()
  // if err != nil {
  //   log.Fatalf("Could not download file id %s: %v", id, err)
  // }
  // defer res.Body.Close()
  // // TODO: Somehow, the download seems to get truncated at 32kB;
  // //       find out how to fix this.
  // p := make([]byte, 1024*1024*1024 + 1)
  // n, err := res.Body.Read(p)
  // p = p[0:n]
  // fmt.Printf("Read %d bytes from response", n)
  // if err != nil {
  //   fmt.Printf(": %v", err)
  // }
  // fmt.Println("")
  // fmt.Println("Save file as:")
  // var name string
  // if _, err := fmt.Scan(&name); err != nil {
  //   log.Fatalf("Unable to get filename to save contents")
  // }
  // of, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
  // if err != nil {
  //   log.Fatalf("Unable to open %s for writing: %v", name, err)
  // }
  // if n, err := of.Write(p); err != nil || n != len(p) {
  //   log.Fatalf("Only wrote %d bytes of of %d: %v", n, len(p), err)
  // }

  // if err != nil {
  //   log.Fatalf("Could not get current user: %v", err)
  // }
  // nf := &drive.File{
  //   Description: "hello world upload",
  //   MimeType: "application/vnd.google-apps.document",
  //   Name: "Hello World Doc",
  // }
  // nf2, err := srv.Files.Create(nf).Media(strings.NewReader("Hello, <Eric was here> world!\n")).Do()
  // if err != nil {
  //   log.Fatalf("Failed to create doc: %v", err)
  // }
  // fmt.Println("%v", nf2)

  rdr := strings.NewReader(data.Contents)

  if _, err := srv.Files.Update(data.DocId, nil).Media(rdr).Do(); err != nil {
    log.Fatalf("Failed to update doc: %v", err)
  }
  fmt.Println("Update pushed")

  // abt, err := srv.About.Get().Fields("importFormats").Do()
  // if err != nil {
  //   log.Fatalf("Failed to retrieve About: %v", err)
  // }
  // fmt.Printf("About:\n%v\n\n", abt)
  // for k, v := range abt.ImportFormats {
  //   fmt.Printf("%s: %v\n", k, v)
  // }
}
