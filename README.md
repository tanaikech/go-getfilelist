# go-getfilelist v2

[![Build Status](https://travis-ci.org/tanaikech/go-getfilelist.svg?branch=master)](https://travis-ci.org/tanaikech/go-getfilelist)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENCE)

<a name="top"></a>

# Overview

This is a Golang library to retrieve the file list with the folder tree from the specific folder of own Google Drive and shared Drives.

# Description

When I create applications for using Google Drive, I often retrieve a file list from a folder in the application. So far, I had created the script for retrieving a file list from a folder for each application. Recently, I thought that if there is the script for retrieving the file list with the folder tree from the folder of Google Drive as a library, it will be useful for me and other users. So I created this.

## Features

- This library retrieves all files from a folder in Google Drive.
- All files include the folder structure in Google Drive.
- Only folder tree can be also retrieved.

# Install

You can install this using `go get` as follows.

```bash
$ go get -u github.com/tanaikech/go-getfilelist
```

- If "go get" cannot be used, please use this library with `GO111MODULE=on`.

- If you want to use the version 1, please use [here](https://github.com/tanaikech/go-getfilelist/tree/master/v1).

# Method

| Method                         | Explanation                                                                                 |
| :----------------------------- | :------------------------------------------------------------------------------------------ |
| GetFolderTree(\*drive.Service) | Retrieve only folder structure from a folder                                                |
| Do(\*drive.Service)            | Retrieve file list with folder structure from a folder                                      |
| Folder(string)                 | Set folder ID.                                                                              |
| Fields(string)                 | Set fields of files.list of Drive API.                                                      |
| MimeType([]string)             | Set mimeType of files.list of Drive API. By this, you can retrieve files with the mimeType. |

# Usage

There are 3 patterns for using this library.

## 1. Use API key

This is a sample script using API key. When you want to retrieve the API key, please do the following flow.

1. Login to Google.
2. Access to [https://console.cloud.google.com/?hl=en](https://console.cloud.google.com/?hl=en).
3. Click select project at the right side of "Google Cloud Platform" of upper left of window.
4. Click "NEW PROJECT"
   1. Input "Project Name".
   2. Click "CREATE".
   3. Open the created project.
   4. Click "Enable APIs and get credentials like keys".
   5. Click "Library" at left side.
   6. Input "Drive API" in "Search for APIs & Services".
   7. Click "Google Drive API".
   8. Click "ENABLE".
   9. Back to [https://console.cloud.google.com/?hl=en](https://console.cloud.google.com/?hl=en).
   10. Click "Enable APIs and get credentials like keys".
   11. Click "Credentials" at left side.
   12. Click "Create credentials" and select API key.
   13. Copy the API key. You can use this API key.

### Sample script

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    getfilelist "github.com/tanaikech/go-getfilelist"

    "google.golang.org/api/drive/v3"
    "google.golang.org/api/option"
)

func main() {
    APIkey := "### API key ###" // Please set here
    folderID := "### folder ID ###" // Please set here

    ctx := context.Background()
    srv, err := drive.NewService(ctx, option.WithAPIKey(APIkey))
    if err != nil {
      log.Fatalf("Unable to retrieve Drive: %v", err)
    }

    // When you want to retrieve the file list in the folder,
    res, err := getfilelist.Folder(folderID).Fields("files(name,id)").MimeType([]string{"application/pdf", "image/png"}).Do(srv)

    // When you want to retrieve only folder tree in the folder,
    res, err := getfilelist.Folder(folderID).GetFolderTree(srv)

    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    fmt.Println(res)
}
```

### Note

- **When you want to retrieve the file list from the folder using API key, the folder is required to be shared.**
- You can modify the value of `Fields()`. When this is not used, the default fields are used.

## 2. Use OAuth2

Document of OAuth2 is [here](https://developers.google.com/identity/protocols/OAuth2).

### Sample script

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

  getfilelist "github.com/tanaikech/go-getfilelist"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// OAuth2 : Use OAuth2
func getService() *drive.Service {
	ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
	return srv
}

func main() {
    folderID := "### folder ID ###" // Please set here

    srv := getService()

    // When you want to retrieve the file list in the folder,
    res, err := getfilelist.Folder(folderID).Fields("files(name,id)").Do(srv)

    // When you want to retrieve only folder tree in the folder,
    res, err := getfilelist.Folder(folderID).GetFolderTree(srv)

    // When you want to retrieve all file list under root of your Google Drive,
    res, err := getfilelist.Do(srv)

    // When you want to retrieve only folder tree under root of your Google Drive,
    res, err := getfilelist.GetFolderTree(srv)

    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    fmt.Println(res)
}
```

### Note

- Here, as a sample, the script of the authorization uses the script of [quickstart](https://developers.google.com/drive/api/v3/quickstart/go).
- You can modify the value of `Fields()`. When this is not used, the default fields are used.

## 3. Use Service account

Document of Service account is [here](https://developers.google.com/identity/protocols/OAuth2ServiceAccount).

### Sample script

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    getfilelist "github.com/tanaikech/go-getfilelist"

    "google.golang.org/api/drive/v3"
    "google.golang.org/api/option"
)

func main() {
    folderID := "### folder ID ###" // Please set here
    credentialFile := "###" // Please set your credential file of the service account.

    ctx := context.Background()
    srv, err := drive.NewService(ctx, option.WithCredentialsFile(credentialFile))
    if err != nil {
      log.Fatal(err)
    }

    // When you want to retrieve the file list in the folder,
    res, err := getfilelist.Folder(folderID).Fields("files(name,id)").Do(srv)

    // When you want to retrieve only folder tree in the folder,
    res, err := getfilelist.Folder(folderID).GetFolderTree(srv)

    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    fmt.Println(res)
}
```

### Note

- You can modify the value of `Fields()`. When this is not used, the default fields are used.

# Values

![](images/downloadFolder_sample.png)

As a sample, when the values are retrieved from above structure, the results of `GetFolderTree()` and `Do()` become as follows.

## Values retrieved by GetFolderTree()

```
res, err := getfilelist.Folder(folderID).GetFolderTree(srv)
```

```json
{
  "id": [
    ["folderIdOfsampleFolder1"],
    ["folderIdOfsampleFolder1", "folderIdOfsampleFolder_2a"],
    ["folderIdOfsampleFolder1", "folderIdOfsampleFolder_2b"],
    [
      "folderIdOfsampleFolder1",
      "folderIdOfsampleFolder_2a",
      "folderIdOfsampleFolder_2a_3a"
    ],
    [
      "folderIdOfsampleFolder1",
      "folderIdOfsampleFolder_2b",
      "folderIdOfsampleFolder_2b_3a"
    ],
    [
      "folderIdOfsampleFolder1",
      "folderIdOfsampleFolder_2b",
      "folderIdOfsampleFolder_2b_3b"
    ],
    [
      "folderIdOfsampleFolder1",
      "folderIdOfsampleFolder_2b",
      "folderIdOfsampleFolder_2b_3b",
      "folderIdOfsampleFolder_2b_3b_4a"
    ]
  ],
  "names": [
    "sampleFolder1",
    "sampleFolder_2a",
    "sampleFolder_2b",
    "sampleFolder_2a_3a",
    "sampleFolder_2b_3a",
    "sampleFolder_2b_3b",
    "sampleFolder_2b_3b_4a"
  ],
  "folders": [
    "folderIdOfsampleFolder1",
    "folderIdOfsampleFolder_2a",
    "folderIdOfsampleFolder_2b",
    "folderIdOfsampleFolder_2a_3a",
    "folderIdOfsampleFolder_2b_3a",
    "folderIdOfsampleFolder_2b_3b",
    "folderIdOfsampleFolder_2b_3b_4a"
  ]
}
```

## Values retrieved by Do()

```
res, err := getfilelist.Folder(folderID).Fields("files(name,mimeType)").Do(srv)
```

```json
{
  "searchedFolder": {
    "id": "###",
    "name": "sampleFolder1",
    "mimeType": "application/vnd.google-apps.folder",
    "parents": ["###"],
    "createdTime": "2000-01-01T01:23:45.000Z",
    "modifiedTime": "2000-01-01T01:23:45.000Z",
    "webViewLink": "https://drive.google.com/drive/folders/###",
    "owners": [
      { "displayName": "###", "permissionId": "###", "emailAddress": "###" }
    ],
    "shared": true
  },
  "folderTree": {
    "id": [
      ["folderIdOfsampleFolder1"],
      ["folderIdOfsampleFolder1", "folderIdOfsampleFolder_2a"],
      ["folderIdOfsampleFolder1", "folderIdOfsampleFolder_2b"],
      [
        "folderIdOfsampleFolder1",
        "folderIdOfsampleFolder_2a",
        "folderIdOfsampleFolder_2a_3a"
      ],
      [
        "folderIdOfsampleFolder1",
        "folderIdOfsampleFolder_2b",
        "folderIdOfsampleFolder_2b_3a"
      ],
      [
        "folderIdOfsampleFolder1",
        "folderIdOfsampleFolder_2b",
        "folderIdOfsampleFolder_2b_3b"
      ],
      [
        "folderIdOfsampleFolder1",
        "folderIdOfsampleFolder_2b",
        "folderIdOfsampleFolder_2b_3b",
        "folderIdOfsampleFolder_2b_3b_4a"
      ]
    ],
    "names": [
      "sampleFolder1",
      "sampleFolder_2a",
      "sampleFolder_2b",
      "sampleFolder_2a_3a",
      "sampleFolder_2b_3a",
      "sampleFolder_2b_3b",
      "sampleFolder_2b_3b_4a"
    ],
    "folders": [
      "folderIdOfsampleFolder1",
      "folderIdOfsampleFolder_2a",
      "folderIdOfsampleFolder_2b",
      "folderIdOfsampleFolder_2a_3a",
      "folderIdOfsampleFolder_2b_3a",
      "folderIdOfsampleFolder_2b_3b",
      "folderIdOfsampleFolder_2b_3b_4a"
    ]
  },
  "fileList": [
    {
      "folderTree": ["folderIdOfsampleFolder1"],
      "files": [
        {
          "name": "Spreadsheet1",
          "mimeType": "application/vnd.google-apps.spreadsheet"
        }
      ]
    },
    {
      "folderTree": ["folderIdOfsampleFolder1", "folderIdOfsampleFolder_2a"],
      "files": [
        {
          "name": "Spreadsheet2",
          "mimeType": "application/vnd.google-apps.spreadsheet"
        }
      ]
    },
    {
      "folderTree": ["folderIdOfsampleFolder1", "folderIdOfsampleFolder_2b"],
      "files": [
        {
          "name": "Spreadsheet4",
          "mimeType": "application/vnd.google-apps.spreadsheet"
        }
      ]
    },
    {
      "folderTree": [
        "folderIdOfsampleFolder1",
        "folderIdOfsampleFolder_2a",
        "folderIdOfsampleFolder_2a_3a"
      ],
      "files": null
    },
    {
      "folderTree": [
        "folderIdOfsampleFolder1",
        "folderIdOfsampleFolder_2b",
        "folderIdOfsampleFolder_2b_3a"
      ],
      "files": [
        {
          "name": "Spreadsheet3",
          "mimeType": "application/vnd.google-apps.spreadsheet"
        }
      ]
    },
    {
      "folderTree": [
        "folderIdOfsampleFolder1",
        "folderIdOfsampleFolder_2b",
        "folderIdOfsampleFolder_2b_3b"
      ],
      "files": null
    },
    {
      "folderTree": [
        "folderIdOfsampleFolder1",
        "folderIdOfsampleFolder_2b",
        "folderIdOfsampleFolder_2b_3b",
        "folderIdOfsampleFolder_2b_3b_4a"
      ],
      "files": [
        {
          "name": "Document1",
          "mimeType": "application/vnd.google-apps.document"
        },
        {
          "name": "image1.png",
          "mimeType": "image/png"
        },
        {
          "name": "Slides1",
          "mimeType": "application/vnd.google-apps.presentation"
        },
        {
          "name": "Spreadsheet5",
          "mimeType": "application/vnd.google-apps.spreadsheet"
        },
        {
          "name": "StandaloneProject1",
          "mimeType": "application/vnd.google-apps.script"
        },
        {
          "name": "Test1.txt",
          "mimeType": "text/plain"
        }
      ]
    }
  ],
  "totalNumberOfFiles": 10,
  "totalNumberOfFolders": 7
}
```

# For other languages

As the libraries "GetFileList" for other languages, there are following libraries.

- Golang: [https://github.com/tanaikech/go-getfilelist](https://github.com/tanaikech/go-getfilelist)
- Google Apps Script: [https://github.com/tanaikech/FilesApp](https://github.com/tanaikech/FilesApp)
- Javascript: [https://github.com/tanaikech/GetFileList_js](https://github.com/tanaikech/GetFileList_js)
- Node.js: [https://github.com/tanaikech/node-getfilelist](https://github.com/tanaikech/node-getfilelist)
- Python: [https://github.com/tanaikech/getfilelistpy](https://github.com/tanaikech/getfilelistpy)

---

<a name="licence"></a>

# Licence

[MIT](LICENCE)

<a name="Author"></a>

# Author

[Tanaike](https://tanaikech.github.io/about/)

If you have any questions and commissions for me, feel free to tell me.

<a name="Update_History"></a>

# Update History

- v1.0.0 (November 2, 2018)

  1. Initial release.

- v1.0.1 (November 13, 2018)

  1. From this version, in order to retrieve files and file information, "google.golang.org/api/drive/v3" is used.
     - By this, when the values are retrieved from this library, users can use the structure of `drive.File`.
     - Script using this library can be seen at [goodls](https://github.com/tanaikech/goodls).

- v1.0.2 (December 12, 2018)

  1. New method for selecting mimeType was added. When this method is used, files with the specific mimeType in the specific folder can be retrieved.

- v1.0.3 (May 14, 2020)

  1. Shared drive got to be able to be used. The file list can be retrieved from both your Google Drive and the shared drive.

     - For example, when the folder ID in the shared Drive is used `folderID` of `Folder(folderID)`, you can retrieve the file list from the folder in the shared Drive.

- v1.0.4 (June 1, 2020)

  1. When the file is retrieved from the shared drive, the parameter was not completed. This bug was removed.

- v2.0.0 (February 25, 2022)

  1. By changing the specification of methods, `drive.New()` and `transport.APIKey` were deprecated. By this, I updated the specification of this library as the version 2. When you are required to continue to use `drive.New()` and `transport.APIKey`, you can use them with the version 1. At version 1, `*http.Client` is given to this library. From version 2, `*drive.Service` is given instead of `*http.Client`. Please be careful this. You can see the sample script at the above.

[TOP](#top)
