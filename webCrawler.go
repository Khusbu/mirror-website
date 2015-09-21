package main

import (
    "fmt"
    "flag"
    "os"
    "net/http"
    "net/url"
    "io"
    "golang.org/x/net/html"
)

func retrieve(uri string){

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
    dir_path := parsed_url.Host+parsed_url.Path
    dir := os.MkdirAll(dir_path, 0777)

    if(dir != nil){
        fmt.Println("Directory Create Error: ",dir)
        os.Exit(1)
    }

    fileWriter, err := os.Create(dir_path+"file.txt")

    if(err != nil){
        fmt.Println("File Open Error: ",err)
        os.Exit(1)
    }

    resp_reader := io.TeeReader(resp.Body, fileWriter)

    z := html.NewTokenizer(resp_reader)
    countLinks := 0
    for{
        tt := z.Next();
        switch{
            case tt==html.ErrorToken:
                return
            case tt==html.StartTagToken:
                t := z.Token()

                if t.Data == "a"{
                    countLinks++
                    for _,a := range t.Attr{
                        if a.Key == "href"{
                            fmt.Println("Link: ", a.Val)
                            break;
                        }

                    }

                }
        }
    }
}

func main(){
    flag.Parse()
    args := flag.Args()

    if(len(args)<1){
        fmt.Println("Specify a start page")
        os.Exit(1)
     }
    retrieve(args[0])
}
