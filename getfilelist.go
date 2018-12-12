// Package getfilelist (getfilelist.go) :
// This is a Golang library to retrieve the file list with the folder tree from the specific folder of Google Drive.
package getfilelist

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

const (
	defFields = "files(createdTime,description,id,mimeType,modifiedTime,name,owners,parents,permissions,shared,size,webContentLink,webViewLink),nextPageToken"
)

// BaseInfo : Base information
type BaseInfo struct {
	Client           *http.Client
	CustomFields     string
	FolderID         string
	InputtedMimeType []string
	SearchFolder     *drive.File
	Srv              *drive.Service
}

// FileListDl : Retrieved file list.
type FileListDl struct {
	SearchedFolder       *drive.File   `json:"searchedFolder,omitempty"`
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
	FolderTree []string      `json:"folderTree"`
	Files      []*drive.File `json:"files"`
}

// fileListSt : File list.
type fileListSt struct {
	NextPageToken string
	Files         []*drive.File
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
	var mType string
	if len(b.InputtedMimeType) > 0 {
		for i, e := range b.InputtedMimeType {
			if i == 0 {
				mType = " and (mimeType='" + e + "'"
				continue
			}
			mType += " or mimeType='" + e + "'"
		}
		mType += ")"
	}
	for i, id := range FolderTree.Folders {
		q := "'" + id + "' in parents and mimeType != 'application/vnd.google-apps.folder' and trashed=false" + mType
		fm, err := b.getListLoop(q, fields)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
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

// getList : For retrieving file list.
func (b *BaseInfo) getList(ptoken, q, fields string) (*drive.FileList, error) {
	f := []googleapi.Field{"nextPageToken", googleapi.Field(fields)}
	r, err := b.Srv.Files.List().PageSize(1000).PageToken(ptoken).OrderBy("name").Q(q).Fields(f...).Do()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// getListLoop : Loop for retrieving file list.
func (b *BaseInfo) getListLoop(q, fields string) (*fileListSt, error) {
	f := &fileListSt{}
	nextPageToken := ""
	for {
		body, err := b.getList(nextPageToken, q, fields)
		if err != nil {
			return nil, err
		}
		f.Files = append(f.Files, body.Files...)
		if body.NextPageToken == "" {
			break
		}
		nextPageToken = body.NextPageToken
	}
	return f, nil
}

// getAllfoldersRecursively : Recursively get folder tree using Drive API.
func (b *BaseInfo) getAllfoldersRecursively(id string, parents []string, fls *folderTr) *folderTr {
	q := "'" + id + "' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false"
	fields := "files(id,mimeType,name,parents,size),nextPageToken"
	fm, err := b.getListLoop(q, fields)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	var temp forFTTemp
	for _, e := range fm.Files {
		ForFt := &forFT{
			ID:   e.Id,
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
func createFolderTreeID(fm *fileListSt, id string, parents []string, fls *folderTr) *folderTr {
	var temp forFTTemp
	for _, e := range fm.Files {
		if len(e.Parents) > 0 && e.Parents[0] == id {
			ForFt := &forFT{
				ID:   e.Id,
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

// getFolderByFolder : Retrieve folder tree by folder by folder.
func (b *BaseInfo) getFolderByFolder() *FolderTree {
	tr := &folderTr{Search: b.SearchFolder.Id}
	return b.getAllfoldersRecursively(b.SearchFolder.Id, []string{}, tr).getDlFoldersS(b.SearchFolder.Name)
}

// getFromFolders : Retrieve folder tree from all folders.
func (b *BaseInfo) getFromAllFolders() *FolderTree {
	q := "mimeType='application/vnd.google-apps.folder' and trashed=false"
	fields := "files(id,mimeType,name,parents,size),nextPageToken"
	fm, err := b.getListLoop(q, fields)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	tr := &folderTr{Search: b.SearchFolder.Id}
	return createFolderTreeID(fm, b.SearchFolder.Id, []string{}, tr).getDlFoldersS(b.SearchFolder.Name)
}

// getFileInf : Retrieve file infomation using Drive API.
func (b *BaseInfo) getFileInf() error {
	fields := []googleapi.Field{"createdTime,id,mimeType,modifiedTime,name,owners,parents,shared,webContentLink,webViewLink"}
	res, err := b.Srv.Files.Get(b.FolderID).Fields(fields...).Do()
	if err != nil {
		return err
	}
	b.SearchFolder = res
	return nil
}

// init : Initialize
func (b *BaseInfo) init() error {
	srv, err := drive.New(b.Client)
	if err != nil {
		return err
	}
	b.Srv = srv
	return nil
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

// MimeType : Set mimeType
func (b *BaseInfo) MimeType(mimeType []string) *BaseInfo {
	b.InputtedMimeType = mimeType
	return b
}

// GetFolderTree : Retrieve only folder tree under the specific folder.
func (b *BaseInfo) GetFolderTree(client *http.Client) (*FolderTree, error) {
	b.Client = client
	if err := b.init(); err != nil {
		return nil, err
	}
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
	if err := b.init(); err != nil {
		return nil, err
	}
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
	if err := b.init(); err != nil {
		return nil, err
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
	if err := b.init(); err != nil {
		return nil, err
	}
	if err := b.getFileInf(); err != nil {
		return nil, err
	}
	ft := b.getFromAllFolders()
	return b.getFilesFromFolder(ft), nil
}
