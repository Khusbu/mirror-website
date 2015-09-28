package main

import (
    "fmt"
    "flag"
    "os"
    "net/http"
    "net/url"
    "io"
    "golang.org/x/net/html"
    "strings"
    "path/filepath"
)

func createPaths (parsed_url *url.URL) *os.File{
  var dir,file string
  if(strings.Index(parsed_url.Path, ".") >= 0){
   dir, file = filepath.Split(parsed_url.Path)
  } else {
   dir  = parsed_url.Path + "/"
   file = "index.html"
  }

  if(len(dir)>0){
    err := os.MkdirAll(parsed_url.Host + dir, 0777)
    if(err != nil){
        fmt.Println("Directory Create Error: ",dir, err)
        os.Exit(1)
    }
  }
  fileWriter, err := os.Create(parsed_url.Host + dir + file)
  if(err != nil){
      fmt.Println("File Open Error: ",err)
      os.Exit(1)
  }
  return fileWriter
}

func generateLinks(resp_reader io.Reader, parsed_url *url.URL, ch chan string ) {
  z := html.NewTokenizer(resp_reader)
  countLinks := 0
  for{
      tt := z.Next();
      switch{
          case tt==html.ErrorToken:
              if countLinks == 0 {
                
              }
              fmt.Println("Number of links found: ", countLinks)
              return
          case tt==html.StartTagToken:
              t := z.Token()

              if t.Data == "a"{
                  for _, a := range t.Attr{
                      if a.Key == "href"{
                          link, err := url.Parse(a.Val)
                          if err != nil {
                            fmt.Println("Url Parsing Error: ",err)
                            os.Exit(1)
                          }
                          if link.Host == parsed_url.Host{
                            countLinks++
                            ch <- a.Val
                          }
                      }

                  }

              }
        }
    }
}

func retrieve(uri string, ch chan string){
    parsed_url, err := url.Parse(uri)
    if(err != nil){
      fmt.Println("Url Parsing Error: ",err)
      os.Exit(1)
    }
    resp, err := http.Get(uri)

    if(err != nil){
        fmt.Println("Http Transport Error: ",err)
        os.Exit(1)
    }
    fileWriter := createPaths(parsed_url)
    resp_reader := io.TeeReader(resp.Body, fileWriter)
    defer fileWriter.Close()
    generateLinks(resp_reader, parsed_url, ch)

}

func main(){
    flag.Parse()
    args := flag.Args()

    if(len(args)<1){
        fmt.Println("Specify a start page")
        os.Exit(1)
     }
     ch := make(chan string, 10)
     ch <- args[0]
     for uri := range ch{
       go retrieve(uri, ch)
     }
}
