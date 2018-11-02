// Package getfilelist (getfilelist.go) :
// This is a Golang library to retrieve the file list with the folder tree from the specific folder of Google Drive.
package getfilelist

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	defFields = "files(createdTime,description,id,mimeType,modifiedTime,name,owners,parents,permissions,shared,size,webContentLink,webViewLink),nextPageToken"
)

// BaseInfo : Base information
type BaseInfo struct {
	Client       *http.Client
	CustomFields string
	FolderID     string
	SearchFolder FileS
}

// FileListDl : Retrieved file list.
type FileListDl struct {
	SearchedFolder       FileS         `json:"searchedFolder,omitempty"`
	FolderTree           *FolderTree   `json:"folderTree,omitempty"`
	FileList             []FileListEle `json:"fileList,omitempty"`
	TotalNumberOfFiles   int64         `json:"totalNumberOfFiles,omitempty"`
	TotalNumberOfFolders int64         `json:"totalNumberOfFolders,omitempty"`
}

// FolderTree : Struct for folder tree.
type FolderTree struct {
	IDs     [][]string `json:"id,omitempty"`
	Names   []string   `json:"names,omitempty"`
	Folders []string   `json:"folders,omitempty"`
}

// FileListEle : Struct for file list.
type FileListEle struct {
	FolderTree []string `json:"folderTree"`
	Files      []FileS  `json:"files"`
}

// FileS : Structure of a file information.
type FileS struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name,omitempty"`
	MimeType     string   `json:"mimeType,omitempty"`
	Parents      []string `json:"parents,omitempty"`
	CreatedTime  string   `json:"createdTime,omitempty"`
	ModifiedTime string   `json:"modifiedTime,omitempty"`
	Size         string   `json:"size,omitempty"`
	WebLink      string   `json:"webContentLink,omitempty"`
	WebView      string   `json:"webViewLink,omitempty"`
	Owners       []struct {
		Name         string `json:"displayName,omitempty"`
		PermissionID string `json:"permissionId"`
		Email        string `json:"emailAddress,omitempty"`
	} `json:"owners,omitempty"`
	Shared      bool `json:"shared,omitempty"`
	Permissions []struct {
		Kind         string `json:"kind"`
		ID           string `json:"id"`
		Type         string `json:"type"`
		EmailAddress string `json:"emailAddress"`
		Role         string `json:"role"`
		DisplayName  string `json:"displayName"`
		PhotoLink    string `json:"photoLink"`
		Deleted      bool   `json:"deleted"`
	} `json:"permissions,omitempty"`
}

// fileListSt : File list.
type fileListSt struct {
	NextPageToken string
	Files         []FileS
}

// forFT : For creating folder tree.
type forFT struct {
	Name   string
	ID     string
	Parent string
	Tree   []string
}

// folderTr : For creating folder tree.
type folderTr struct {
	Temp   [][]forFT
	Search string
}

// forFTTemp : For creating folder tree.
type forFTTemp struct {
	Temp []forFT
}

// fetch : fetch
func (b *BaseInfo) fetch(url string) ([]byte, error) {
	res, _ := b.Client.Get(url)
	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%s", r)
	}
	defer res.Body.Close()
	return r, nil
}

// getFileInf : Retrieve file infomation using Drive API.
func (b *BaseInfo) getFileInf() error {
	url := "https://www.googleapis.com/drive/v3/files/" + b.FolderID + "?fields=createdTime%2Cid%2CmimeType%2CmodifiedTime%2Cname%2Cowners%2Cparents%2Cshared%2CwebContentLink%2CwebViewLink"
	res, err := b.fetch(url)
	if err != nil {
		return err
	}
	json.Unmarshal(res, &b.SearchFolder)
	return nil
}

// getList : For retrieving file list.
func (b *BaseInfo) getList(ptoken, q, fields string) ([]byte, error) {
	number := 1000
	tokenparams := url.Values{}
	tokenparams.Set("orderBy", "name")
	tokenparams.Set("pageSize", strconv.Itoa(number))
	tokenparams.Set("q", q)
	tokenparams.Set("fields", fields)
	if len(ptoken) > 0 {
		tokenparams.Set("pageToken", ptoken)
	}
	url := "https://www.googleapis.com/drive/v3/files?" + tokenparams.Encode()
	body, err := b.fetch(url)
	if err != nil {
		return nil, err
	}
	return body, err
}

// getListLoop : Loop for retrieving file list.
func (b *BaseInfo) getListLoop(q, fields string) fileListSt {
	var fm fileListSt
	var fl fileListSt
	var dmy fileListSt
	fm.NextPageToken = ""
	for {
		body, err := b.getList(fm.NextPageToken, q, fields)
		json.Unmarshal(body, &fl)
		fm.NextPageToken = fl.NextPageToken
		fm.Files = append(fm.Files, fl.Files...)
		fl.NextPageToken = ""
		fl.Files = dmy.Files
		if len(fm.NextPageToken) == 0 || err != nil {
			break
		}
	}
	return fm
}

// getDlFoldersS : Retrieve each folder from folder list using folder ID. This is for shared folders.
func (fr *folderTr) getDlFoldersS(searchFolderName string) *FolderTree {
	fT := &FolderTree{}
	fT.Folders = append(fT.Folders, fr.Search)
	fT.Names = append(fT.Names, searchFolderName)
	fT.IDs = append(fT.IDs, []string{fr.Search})
	for _, e := range fr.Temp {
		for _, f := range e {
			fT.Folders = append(fT.Folders, f.ID)
			var tmp []string
			tmp = append(tmp, f.Tree...)
			tmp = append(tmp, f.ID)
			fT.IDs = append(fT.IDs, tmp)
			fT.Names = append(fT.Names, f.Name)
		}
	}
	return fT
}

// getFilesFromFolder : Retrieve file list from folder list.
func (b *BaseInfo) getFilesFromFolder(FolderTree *FolderTree) *FileListDl {
	f := &FileListDl{}
	f.SearchedFolder = b.SearchFolder
	f.FolderTree = FolderTree
	fields := func() string {
		if b.CustomFields == "" {
			return defFields
		}
		if !strings.Contains(b.CustomFields, "nextPageToken") {
			b.CustomFields += ",nextPageToken"
		}
		return b.CustomFields
	}()
	for i, id := range FolderTree.Folders {
		q := "'" + id + "' in parents and mimeType != 'application/vnd.google-apps.folder' and trashed=false"
		fm := b.getListLoop(q, fields)
		var fe FileListEle
		fe.FolderTree = FolderTree.IDs[i]
		fe.Files = append(fe.Files, fm.Files...)
		f.FileList = append(f.FileList, fe)
	}
	f.TotalNumberOfFolders = int64(len(f.FolderTree.Folders))
	f.TotalNumberOfFiles = func() int64 {
		var c int64
		for _, e := range f.FileList {
			c += int64(len(e.Files))
		}
		return c
	}()
	return f
}

// getAllfoldersRecursively : Recursively get folder tree using Drive API.
func (b *BaseInfo) getAllfoldersRecursively(id string, parents []string, fls *folderTr) *folderTr {
	q := "'" + id + "' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false"
	fields := "files(id,mimeType,name,parents,size),nextPageToken"
	fm := b.getListLoop(q, fields)
	var temp forFTTemp
	for _, e := range fm.Files {
		ForFt := &forFT{
			ID:   e.ID,
			Name: e.Name,
			Parent: func() string {
				if len(e.Parents) > 0 {
					return e.Parents[0]
				}
				return ""
			}(),
			Tree: append(parents, id),
		}
		temp.Temp = append(temp.Temp, *ForFt)
	}
	if len(temp.Temp) > 0 {
		fls.Temp = append(fls.Temp, temp.Temp)
		for _, e := range temp.Temp {
			b.getAllfoldersRecursively(e.ID, e.Tree, fls)
		}
	}
	return fls
}

// createFolderTreeID : Create a folder tree.
func createFolderTreeID(fm fileListSt, id string, parents []string, fls *folderTr) *folderTr {
	var temp forFTTemp
	for _, e := range fm.Files {
		if len(e.Parents) > 0 && e.Parents[0] == id {
			ForFt := &forFT{
				ID:   e.ID,
				Name: e.Name,
				Parent: func() string {
					if len(e.Parents) > 0 {
						return e.Parents[0]
					}
					return ""
				}(),
				Tree: append(parents, id),
			}
			temp.Temp = append(temp.Temp, *ForFt)
		}
	}
	if len(temp.Temp) > 0 {
		fls.Temp = append(fls.Temp, temp.Temp)
		for _, e := range temp.Temp {
			createFolderTreeID(fm, e.ID, e.Tree, fls)
		}
	}
	return fls
}

// getFolderByFolder : Retrieve folder tree by folder by folder.
func (b *BaseInfo) getFolderByFolder() *FolderTree {
	tr := &folderTr{Search: b.SearchFolder.ID}
	return b.getAllfoldersRecursively(b.SearchFolder.ID, []string{}, tr).getDlFoldersS(b.SearchFolder.Name)
}

// getFromFolders : Retrieve folder tree from all folders.
func (b *BaseInfo) getFromAllFolders() *FolderTree {
	q := "mimeType='application/vnd.google-apps.folder' and trashed=false"
	fields := "files(id,mimeType,name,parents,size),nextPageToken"
	fm := b.getListLoop(q, fields)
	tr := &folderTr{Search: b.SearchFolder.ID}
	return createFolderTreeID(fm, b.SearchFolder.ID, []string{}, tr).getDlFoldersS(b.SearchFolder.Name)
}

// Fields : Set fields for file list.
func (b *BaseInfo) Fields(fields string) *BaseInfo {
	b.CustomFields = fields
	return b
}

// Folder : Set folder ID
func Folder(folderID string) *BaseInfo {
	b := &BaseInfo{
		FolderID: folderID,
	}
	return b
}

// GetFolderTree : Retrieve only folder tree under the specific folder.
func (b *BaseInfo) GetFolderTree(client *http.Client) (*FolderTree, error) {
	b.Client = client
	if err := b.getFileInf(); err != nil {
		return nil, err
	}
	if b.SearchFolder.Shared {
		ft := b.getFolderByFolder()
		return ft, nil
	}
	ft := b.getFromAllFolders()
	return ft, nil
}

// Do : Retrieve all file list and folder tree under the specific folder.
func (b *BaseInfo) Do(client *http.Client) (*FileListDl, error) {
	b.Client = client
	if err := b.getFileInf(); err != nil {
		return nil, err
	}
	if b.SearchFolder.Shared {
		ft := b.getFolderByFolder()
		return b.getFilesFromFolder(ft), nil
	}
	ft := b.getFromAllFolders()
	return b.getFilesFromFolder(ft), nil
}

// GetFolderTree : Retrieve only folder tree under root.
func GetFolderTree(client *http.Client) (*FolderTree, error) {
	b := &BaseInfo{
		Client:   client,
		FolderID: "root",
	}
	if err := b.getFileInf(); err != nil {
		return nil, err
	}
	ft := b.getFromAllFolders()
	return ft, nil
}

// Do : Retrieve all file list and folder tree under root.
func Do(client *http.Client) (*FileListDl, error) {
	b := &BaseInfo{
		Client:   client,
		FolderID: "root",
	}
	if err := b.getFileInf(); err != nil {
		return nil, err
	}
	ft := b.getFromAllFolders()
	return b.getFilesFromFolder(ft), nil
}
