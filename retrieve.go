package main

import (
    "fmt"
    "net/http"
    "io/ioutil"
)

func main(){
    resp, err := http.Get("http://www.google.com")

    fmt.Println("Error: ",err)

    body,err := ioutil.ReadAll(resp.Body)

    fmt.Println(err)
    fmt.Println(string(body))
}
