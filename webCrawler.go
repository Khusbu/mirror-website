package main

import (
    "fmt"
    "flag"
    "os"
    "net/http"
    "net/url"
    "io"
  //  "io/ioutil"
    "golang.org/x/net/html"
    "strings"
    "path/filepath"
  //  "log"
    "sync"
)

const MAX_GO_ROUTINE = 100
var start_url *url.URL
var (
    visited = make(map[string]bool)
    visited_mutex sync.Mutex
)
var (
    queue = make([]string, 0)
    push_mutex sync.Mutex
    pop_mutex sync.Mutex
)
//var file_paths = make(map[string]string)
//var relative = make(map[string]string)

//Create path for response file

func createPaths(parsed_url *url.URL) (*os.File, string){
    var dir,file string
    dir, file = filepath.Split(parsed_url.Path)
    if file == "" {
        file = "index.html"
    } 
    if parsed_url.RawQuery != ""{
        file += "?" + parsed_url.RawQuery
    } 
    err := os.MkdirAll(parsed_url.Host + dir, 0777)
    if(err != nil){
        fmt.Println("Directory Create Error: ",dir, err)
        return nil, ""
    }
    if !strings.HasSuffix(dir, "/"){
        dir = dir + "/"
    }
    file_path := parsed_url.Host + dir + file
    fileWriter, err := os.Create(file_path)
    if(err != nil){
        fmt.Println("File Open Error: ",err)
        return nil, ""
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
    if data == "img" || uri.Host == start_url.Host {
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
    //Fetch links from response
func generateLinks(resp_reader io.Reader,  uri *url.URL) {
    z := html.NewTokenizer(resp_reader)
    countLinks := 0
    for{
        tt := z.Next()
        switch tt {
        case html.ErrorToken:
            //fmt.Println("Number of links: ", countLinks)
            return
        case html.StartTagToken, html.SelfClosingTagToken:
            t := z.Token()
            if t.Data == "a" || t.Data == "link" || t.Data == "img" {
                for _, a := range t.Attr{
                    if (a.Key == "href" || a.Key == "src") && !strings.Contains(a.Val, "#"){
                        countLinks += fixUrl(a.Val, uri, t.Data)
                     }
                }
            }
        }
    }
}

    //Retrieves a link

func retrieve(uri string, syncChan chan int){
    fmt.Println("Fetching:  ", uri)
    resp, err := http.Get(uri)

    if(err != nil){
        fmt.Println("Http Transport Error: ", uri, "     ", err)
        return
    }
    defer resp.Body.Close()

    actual_url := resp.Request.URL
    if uri != actual_url.String() {
        write_visited(actual_url.String())
    }
    fileWriter, file_path := createPaths(actual_url)
    if fileWriter != nil && file_path != "" {
        // file_paths[uri] = file_path
        resp_reader := io.TeeReader(resp.Body, fileWriter)
        defer fileWriter.Close()

        generateLinks(resp_reader, actual_url)
    }
    <-syncChan
}
/*
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
}*/

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
        fmt.Println("Provide full url like http://www.example.com and try again!")
        os.Exit(1)
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

func main() {
    flag.Parse()
    args := flag.Args()
    if len(args)<1 {
        fmt.Println("Specify a start url")
        os.Exit(1)
     }
     fix_start_url(args[0])
     syncChan := make(chan int, MAX_GO_ROUTINE)
     push(start_url.String())

     for {
         fmt.Println("Queue: ",len(queue))
         fmt.Println("Threads: ",len(syncChan))
         if len(queue) > 0 {
           current_url := pop()
           if !read_visited(current_url) {
              syncChan <- 1
              write_visited(current_url)
              go retrieve(current_url, syncChan) 
           }
         }
         if len(syncChan) == 0 && len(queue) == 0 {
            break
         }
     }
   //postProcessing()
}
