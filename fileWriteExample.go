package main

import (
        "io/ioutil"
        "log"
        "strings"
        "path/filepath"
        "os"
        "fmt"
)

func walkFn(path string, info os.FileInfo, err error) error {
    input, err := ioutil.ReadFile(path)
    if err == nil {
      lines := strings.Split(string(input), "\n")

      for i, line := range lines {
              if strings.Contains(line, "LOL") {
                      lines[i] = "absolute"
              }
      }
      output := strings.Join(lines, "\n")
      err = ioutil.WriteFile(path, []byte(output), 0644)
      if err != nil {
              log.Fatalln(err)
      }
  }
  return nil
}

func main() {
        err := filepath.Walk("/home/administrator/Desktop/Programs/GoRepositories/src/Mirror-Website/abc", walkFn)
        fmt.Println(err)
}
