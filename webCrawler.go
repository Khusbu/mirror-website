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
    "sync"
    //"time"
)

const MAX_GO_ROUTINE = 100
var start_url *url.URL
var (
    visited = make(map[string]bool)
    visited_mutex sync.Mutex
)
var (
    thread_count = 0
    thread_count_mutex sync.Mutex
)
var (
    queue = make([]string, 0)
    push_mutex sync.Mutex
    pop_mutex sync.Mutex
)
var file_paths = make(map[string]string)
var relative = make(map[string]string)

// func isFileExists (string path) {
//   if finfo, err := os.Stat(dir); err == nil {
//     if !finfo.IsDir() {
//       return true;
//     }
//   }
//   return false;
// }

func createPaths(parsed_url *url.URL) (*os.File, string){
  var dir,file string
  if parsed_url.RawQuery != ""{
    dir, file = filepath.Split(parsed_url.Path+"?"+parsed_url.RawQuery)
  } else {
    dir, file = filepath.Split(parsed_url.Path)
  }
  if file == "" {
      file = "index.html"
  } else if filepath.Ext(file) == ""{
    file += ".html"
  }
  //if(len(dir)>0) {
    err := os.MkdirAll(parsed_url.Host + dir, 0777)
    if(err != nil){
          fmt.Println("Directory Create Error: ",dir, err)
          os.Exit(1)
    }
  //}
  if !strings.HasSuffix(dir, "/"){
      dir = dir + "/"
  }
  file_path := parsed_url.Host + dir + file
  fileWriter, err := os.Create(file_path)
  if(err != nil){
      fmt.Println("File Open Error: ",err)
      os.Exit(1)
  }
  return fileWriter, file_path
}


//Converts relative links to absolute links

func fixUrl(href string, baseUrl *url.URL, data string) int {
   link, err := url.Parse(href)
    if err!= nil{
        fmt.Println("Parsing relative link Error: ", err)
        return 0//ignoring invalid urls
    }
    uri := baseUrl.ResolveReference(link)
    if data == "img" || data == "link" || uri.Host == start_url.Host {
        absolute_link := uri.String()
        if !read_visited(absolute_link) {
            push(absolute_link)
        /*    if strings.HasSuffix(href,"/") {
                file_paths[href] = href + "index.html"
            }
            if link.IsAbs(){
                relative[href] = absolute_link
            }*/
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
              //fmt.Println("Number of links: ", countLinks)
              return
          case tt==html.StartTagToken:
              t := z.Token()
              if t.Data == "a" || t.Data == "link" {
                  for _, a := range t.Attr{
                      if a.Key == "href" && !strings.Contains(a.Val, "#"){
                          countLinks += fixUrl(a.Val, uri, t.Data)
                     }
                  }
              }
          case tt==html.SelfClosingTagToken:
              t := z.Token()
              if t.Data == "img" || t.Data == "link" {
                  for _, a := range t.Attr{
                      if a.Key =="src" || a.Key == "href" {
                          countLinks += fixUrl(a.Val, uri, t.Data)
                      }
                  }
              }
        }
    }
}

func retrieveDone(syncChan chan int) {
    change_thread_count("done", syncChan)
}

func retrieve(uri string, syncChan chan int){
    defer retrieveDone(syncChan)

   /* parsed_url, err := url.Parse(uri)
    if err!= nil{
        fmt.Println("Parsing link Error: ", err)
        return
    }*/
    fmt.Println("Fetching:  ", uri)
    resp, err := http.Get(uri)

    if(err != nil){
        fmt.Println("Http Transport Error: ", uri, "     ", err)
        return
    }
    defer resp.Body.Close()
    if uri != resp.Request.URL.String() {
        write_visited(resp.Request.URL.String())
    }
    fileWriter, _ := createPaths(resp.Request.URL)
   // file_paths[uri] = file_path
    resp_reader := io.TeeReader(resp.Body, fileWriter)
    defer fileWriter.Close()

    generateLinks(resp_reader, resp.Request.URL)
}

func walkFn(path string, info os.FileInfo, err error) error {
    if !info.IsDir() {
        input, err := ioutil.ReadFile(path)
        if err != nil{
            log.Fatalln(err)
            return err
        }
        output := string(input)
        dir, _ := filepath.Split(path)
        for rel, abs := range relative{
            rel_url, err := filepath.Rel(dir, file_paths[abs])
            if err != nil{
                log.Fatalln(err)
            }
            output = strings.Replace(output, "\""+rel+"\"", "\""+rel_url+"\"", -1)
            output = strings.Replace(output, "'"+rel+"'", "'"+rel_url+"'", -1)
        }
        err = ioutil.WriteFile(path, []byte(output), 0644)
        if err != nil{
            log.Fatalln(err)
            return err
        }
    }
    return nil
}


func postProcessing(){
  err := filepath.Walk(start_url.Host, walkFn)
   if err != nil{
       log.Fatalln(err)
       return
   }
   fmt.Println("Done!!!")
}
func read_visited(value string)bool {
    visited_mutex.Lock()
    defer visited_mutex.Unlock()
    return visited[value]
}
func write_visited(value string) { 
    visited_mutex.Lock()
    defer visited_mutex.Unlock()
    visited[value] = true
}
func fix_start_url(link string) {
    var err error
    start_url, err = url.Parse(link)
    if err!= nil{
        fmt.Println("Parsing Start Url Error: ",err)
        os.Exit(1)
    }
    if start_url.Scheme == "" {
        start_url.Scheme = "http"
    }
}

func pop() string {
    pop_mutex.Lock()
    defer pop_mutex.Unlock()
    url := queue[0]
    queue = queue[1:]
    return url
}

func push(url string) {
    push_mutex.Lock()
    defer push_mutex.Unlock()
    queue = append(queue, url)
}

func change_thread_count(condition string, syncChan chan int){ 
//         thread_count_mutex.Lock()
//         defer thread_count_mutex.Unlock()
         if condition == "done" {
            <-syncChan
            thread_count--;
         } else if condition == "start" {
             syncChan <- 1
             thread_count++; 
         }
}
func main(){
    flag.Parse()
    args := flag.Args()
    if len(args)<1 {
        fmt.Println("Specify a start page")
        os.Exit(1)
     }
     fix_start_url(args[0])
     syncChan := make(chan int, MAX_GO_ROUTINE)
     push(start_url.String())

     for {
         fmt.Println("Queue: ",len(queue))
         fmt.Println("Count: ",thread_count)
         if len(queue) > 0 {
           current_url := pop()
           if !read_visited(current_url) {
              change_thread_count("start",syncChan)
              write_visited(current_url)
              go retrieve(current_url, syncChan) 
           }
         }
         if thread_count == 0 {
            break
         }
     }
   //postProcessing()
}
