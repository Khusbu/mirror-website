package main

import (
    "fmt"
    "flag"
    "os"
    "net/http"
    "net/url"
    "io"
    "io/ioutil"
    "golang.org/x/net/html"
    "strings"
    "path/filepath"
    "log"
)

var start_url *url.URL
var visited = make(map[string]bool)
var goCount = 0
var file_paths = make(map[string]string)

func createPaths(parsed_url *url.URL) *os.File{
  var dir,file string
  dir, file = filepath.Split(parsed_url.Path)
  if file == "" {
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


//Converts relative links to absolute links

func fixUrl(href string, baseUrl *url.URL) int {
   link, err := url.Parse(href)
    if err!= nil{
        fmt.Println("Parsing relative link Error: ", err)
        return 0//ignoring invalid urls
    }
    uri := baseUrl.ResolveReference(link)
    if uri.Host == start_url.Host {
        absolute_link := uri.String()
        _, ok := visited[absolute_link]
        if !ok {
            visited[absolute_link] = false
            if strings.HasSuffix(href,"/") {
                file_paths[href] = href + "index.html"
            }
            return 1
         }
    }
    return 0
}

func generateLinks(resp_reader io.Reader,  uri *url.URL) {
  z := html.NewTokenizer(resp_reader)
  countLinks := 0
  for{
      tt := z.Next()
      switch{
          case tt==html.ErrorToken:
              fmt.Println("Number of links: ", countLinks)
              return
          case tt==html.StartTagToken:
              t := z.Token()
              if t.Data == "a" || t.Data == "link" {
                  for _, a := range t.Attr{
                      if a.Key == "href" && !strings.Contains(a.Val, "#"){
                          countLinks += fixUrl(a.Val, uri)
                     }
                  }
              }
          case tt==html.SelfClosingTagToken: 
              t := z.Token()
              if t.Data == "img" {
                  for _, a := range t.Attr{
                      if a.Key =="src"{
                          countLinks += fixUrl(a.Val, uri)
                      }
                  }
              }
        }
    }
}

func retrieveDone(syncChan chan int) {
    <-syncChan
    goCount--
}

func retrieve(uri string, syncChan chan int){
    defer retrieveDone(syncChan)

    parsed_url, err := url.Parse(uri)
    if err!= nil{
        fmt.Println("Parsing link Error: ", err)
        os.Exit(1)
    }
    fmt.Println("Fetching:  ", uri)
    resp, err := http.Get(uri)

    if(err != nil){
        fmt.Println("Http Transport Error: ", uri, "     ", err)
        return
    }
    defer resp.Body.Close()

    fileWriter := createPaths(parsed_url)
    resp_reader := io.TeeReader(resp.Body, fileWriter)
    defer fileWriter.Close()

    generateLinks(resp_reader, parsed_url)
}

func walkFn(path string, info os.FileInfo, err error) error {
    if !info.IsDir() {
        input, err := ioutil.ReadFile(path)
        if err != nil{
            log.Fatalln(err)
            return err
        }
        actual_path, err := os.Getwd()
        if err != nil{
            fmt.Println("Error in Getwd: ",err)
            return err
        }
        output := string(input)
        for link, file := range file_paths{
            output = strings.Replace(output, link, file, -1)
        }
        output = strings.Replace(output, "http://"+ start_url.Host, actual_path + "/" + start_url.Host, -1)
        err = ioutil.WriteFile(path, []byte(output), 0644)
        if err != nil{
            log.Fatalln(err)
            return err
        }
    }
    return nil
}


func postProcessing(){
   actual_path, err := os.Getwd()
   if err != nil{
       fmt.Println("Error in Getwd: ",err)
       return
   }
   err = filepath.Walk(actual_path +"/"+ start_url.Host, walkFn)
   if err != nil{
       log.Fatalln(err)
       return
   } 
   fmt.Println("Done!!!")
}

func main(){
    flag.Parse()
    args := flag.Args()
    if(len(args)<1){
        fmt.Println("Specify a start page")
        os.Exit(1)
     }
     var err error
     start_url, err = url.Parse(args[0])
     if err!= nil{
         fmt.Println("Parsing Start Url Error: ",err)
         os.Exit(1)
     }
     if !strings.HasSuffix(args[0], "/"){
        args[0] = args[0] + "/"
     }
     syncChan:= make(chan int, 10)
     visited[args[0]] = false
     for {
        allVisited := true
        for uri, done := range visited {
            if done == true {
                continue
            }
            syncChan <- 1
            visited[uri] = true
            goCount++
            go retrieve(uri, syncChan)
            allVisited = false
            break
        }
        if allVisited == true && goCount == 0 {
            break;
        }
     }
    postProcessing() 
}
