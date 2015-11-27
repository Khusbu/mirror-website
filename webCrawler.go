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
var (
    file_paths = make(map[string]string)
    file_path_mutex sync.Mutex
)
var (
    relative = make(map[string]string) 
    relative_path_mutex sync.Mutex
)
var (
    file_links = make(map[string][]string)
    file_links_mutex sync.Mutex
)
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
    parsed_url_host := ""
    if parsed_url.Host != start_url.Host {
        parsed_url_host = parsed_url.Host
    }
    full_dir_path := start_url.Host +"/"+ parsed_url_host +"/"+ dir + "/"
    err := os.MkdirAll(full_dir_path, 0777)
    if(err != nil){
        fmt.Println("Directory Create Error: ",full_dir_path, err)
        return nil, ""
    }
    file_path := full_dir_path + file
    fileWriter, err := os.Create(file_path)
    if(err != nil){
        fmt.Println("File Open Error: ",err)
        return nil, ""
    }
    full_path := filepath.Dir(file_path) + "/" +file
    return fileWriter, full_path
}

//Stores a map of absolute link as key and fetched link as value

func store_absolute_link(absolute_link string, href string) {
    relative_path_mutex.Lock()
    defer relative_path_mutex.Unlock()
    relative[absolute_link] = href
}

//Converts relative links to absolute links

func fixUrl(href string, baseUrl *url.URL, data string) string {
    link, err := url.Parse(href)
    if err!= nil{
        fmt.Println("Parsing relative link Error: ", err)
        return ""//ignoring invalid urls
    }
    uri := baseUrl.ResolveReference(link)
    if data == "img" || uri.Host == start_url.Host {
        absolute_link := uri.String()
        if !read_visited(absolute_link) {
            push(absolute_link)
            store_absolute_link(absolute_link, href)
        }
        return absolute_link
    }
    return ""
}

func write_file_links(file_pathname string, fetched_links []string) {
    file_links_mutex.Lock()
    defer file_links_mutex.Unlock()
    file_links[file_pathname] = fetched_links
}
//Fetch links from response
func generateLinks(resp_reader io.Reader,  uri *url.URL, file_pathname string) {
    z := html.NewTokenizer(resp_reader)
    fetched_links := make([]string,0)
    for{
        tt := z.Next()
        switch tt {
        case html.ErrorToken:
            write_file_links(file_pathname, fetched_links[0:])
            return
        case html.StartTagToken, html.SelfClosingTagToken:
            t := z.Token()
            if t.Data == "a" || t.Data == "link" || t.Data == "img" {
                for _, a := range t.Attr{
                    if (a.Key == "href" || a.Key == "src") && !strings.Contains(a.Val, "#"){
                        abs_link := fixUrl(a.Val, uri, t.Data)
                        if abs_link != "" {
                            fetched_links = append(fetched_links, abs_link)
                        }
                     }
                }
            }
        }
    }
}

//Stores a map of fetched link as key and filepath as value
func store_file_path(absolute_link string, file_path string) {
    file_path_mutex.Lock()
    defer file_path_mutex.Unlock()
    file_paths[absolute_link] = file_path
}

//Retrieves a link

func retrieve(uri string, syncChan chan int){
    defer func() {
        <-syncChan
    }() 
    fmt.Println("Fetching:  ", uri)
    resp, err := http.Get(uri)

    if(err != nil){
        fmt.Println("Http Transport Error: ", uri, "     ", err)
        return
    }
    defer resp.Body.Close()

    actual_url := resp.Request.URL
    fetched_url, _ := url.Parse(uri)

    if fetched_url.Host == actual_url.Host && resp.StatusCode < 400 {
        fileWriter, file_path := createPaths(actual_url)
        if fileWriter != nil && file_path != "" {
            store_file_path(uri, file_path)
            resp_reader := io.TeeReader(resp.Body, fileWriter)
            generateLinks(resp_reader, actual_url,file_path)
            defer fileWriter.Close()
        }
        if uri != actual_url.String() {
            write_visited(actual_url.String())
        }
    }
}

func walkFn(path string, info os.FileInfo, err error) error {
    if !info.IsDir() {
        input, err := ioutil.ReadFile(path)
        if err != nil{
            fmt.Println("File reading error: ",err)
            return err
        }
        output := string(input)
        dir, _ := filepath.Split(path)
        for _, abs_link := range file_links[path] { 
            rel_url, err := filepath.Rel(dir, file_paths[abs_link])
            if err != nil{
                fmt.Println("Error creating relative path: ",err)
                continue
            }
            output = strings.Replace(output, "\""+relative[abs_link]+"\"", "\""+rel_url+"\"", -1)
            output = strings.Replace(output, "'"+relative[abs_link]+"'", "'"+rel_url+"'", -1)
        }
        err = ioutil.WriteFile(path, []byte(output), 0644)
        if err != nil{
            fmt.Println("Error writing file: ",err)
            return err
        }
    }
    return nil
}


func postProcessing(){
  fmt.Println("Post-processing...")
  err := filepath.Walk(start_url.Host, walkFn)
   if err != nil{
       fmt.Println("Walking error: ", err)
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

func push(href string) {
    push_mutex.Lock()
    defer push_mutex.Unlock()
    queue = append(queue, href)
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
     start_link := start_url.String()
     push(start_link)
     store_absolute_link(start_link, start_link)
     for {
         fmt.Println("Number of URLs in Queue: ",len(queue))
         fmt.Println("Number of Threads running: ",len(syncChan))
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
     if len(file_paths) > 0 {
        postProcessing()
     }
}
