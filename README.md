# deepl-go
This library provides a simple API client function for Deepl.

## Usage
1. Install package
   ```console
   > go get github.com/DaikiYamakawa/deepl-go
   ```
2. We should register valid API key in the environment variable.
    ```console
    > export DEEPL_API_KEY=xxx-xxx-xxx
    ```
3. We can call deepl library in our code.
   ```golang
    package main

    import (
        "context"
        "fmt"

        "github.com/DaikiYamakawa/deepl-go"
    )

    func main() {
        cli, err := deepl.New("https://api.deepl.com", nil)
        if err != nil {
            fmt.Printf("Failed to create client:\n   %+v\n", err)
        }
        translateResponse, err := cli.TranslateSentence(context.Background(), "Hello", "EN", "JA")
        if err != nil {
            fmt.Printf("Failed to translate text:\n   %+v\n", err)
        } else {
            fmt.Printf("%+v\n", translateResponse)
        }
    }
   ```
   ```console
   &{Translations:[{DetectedSourceLanguage:EN Text:こんにちは}]}
   ```