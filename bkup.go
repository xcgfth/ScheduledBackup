package main

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)


type Multimap map[string][]string

type kvs struct {
	key string
	values []string
}

func (multimap Multimap) Add(key, value string) {
	if (len(multimap[key])) == 0 {
		multimap[key] = []string{value}
	} else {
		multimap[key] = append(multimap[key], value)
	}
}

func (multimap Multimap) Get(key string) []string {
	if multimap == nil {
		return nil
	}
	values := multimap[key]
	return values
}


func bkup(path string) (keys []string, time4File map[string]string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		println("This path is not right")
		os.Exit(12)
	}

	time4File = map[string]string{}
	for _, f := range files {
		if !f.IsDir() {

			time4File[f.ModTime().Format(time.RFC3339)]=f.Name()
		}

	}
	keys = []string{}
	for k:= range time4File {

		keys = append(keys, k)
	}
	sort.Strings(keys)
	return
}

func SFTPConn(usr, passwd, host string, port int) (sftpClient *sftp.Client, err error) {
	auth := make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(passwd))
	clientConfig := &ssh.ClientConfig{
		User: usr,
		Auth:auth,
		Timeout:30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	addr := host + ":" + strconv.Itoa(port)
	sshClient , err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		fmt.Println("链接ssh失败", err)
		return
	}
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		fmt.Println("创建客户端失败", err)
		return
	}
	return
}


func uploadFile(SFTPClient *sftp.Client, srcFilePath, destDir, NewFileName string) {
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		fmt.Println("os.open.error:", srcFilePath)
	}
	defer srcFile.Close()
	var DestFileName = path.Base(srcFilePath)

	destFile, err := SFTPClient.Create(path.Join(destDir, NewFileName))
	if err != nil {
		fmt.Println("SFTPClient.Create error:", path.Join(destDir, DestFileName))
	}
	defer destFile.Close()
	ff, err := ioutil.ReadAll(srcFile)
	if err != nil {
		fmt.Println("ReadAll error:", srcFilePath)
	}
	destFile.Write(ff)
	fmt.Println(srcFilePath + " copy file to remote server finished.")
}


func ReserveLatestFiles(SFTPClient *sftp.Client, RemoteDir string, ReserveNumFiles int) {
	var time4File Multimap
	time4File = make(Multimap)
	RemoteFiles, err := SFTPClient.ReadDir(RemoteDir)
	if err != nil {
		fmt.Println("Read remote directory failed. please check the path you give")
	}

	for _, file := range RemoteFiles {
			time4File.Add(file.ModTime().Format(time.RFC3339Nano), file.Name())
	}

	keys := make([]string, 0, len(time4File))
	for k := range time4File {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	totalFiles := len(RemoteFiles)

	count := 0
	for key := range time4File {
		if count <  totalFiles - ReserveNumFiles + 1  {
			for j := 0; j < len(time4File[key]); j++ {
				SFTPClient.Remove(path.Join(RemoteDir, time4File[key][j]))
				count++
			}
		}else {
			break
		}
	}
}


func Call(SrcDir, DestDir, User, Passwd, Host string, Port, ReserveNumFiles int) {
	keys, time4File := bkup(SrcDir)
	latest_file := time4File[keys[len(keys) - 1]] + "_" + keys[len(keys) - 1]
	println(latest_file)
	FileSuffix := path.Ext(time4File[keys[len(keys) - 1]])
	FileName := strings.TrimSuffix(time4File[keys[len(keys) - 1]], FileSuffix)
	NewFileName := FileName + "_" + keys[len(keys) - 1] + FileSuffix
	println(NewFileName)
	srcFilePath := path.Join(SrcDir, time4File[keys[len(keys) - 1]])
	SFTPClient, err := SFTPConn(User, Passwd, Host, Port)
	if err != nil {
		println("Create SFTPClient failed.")
	}
	ReserveLatestFiles(SFTPClient, DestDir, ReserveNumFiles)
	uploadFile(SFTPClient, srcFilePath, DestDir, NewFileName)
}


func main() {
	SrcDir := "E:\\workspaces\\GoWorkspace\\hello\\testdir"
	DestDir := "/data/cgfth/scripts/dir_test"
	User := "user"
	Password := "passwd"
	Host := "host ip"
	Port := 22
	ReserveNumFiles := 5
	for {
		Call(SrcDir, DestDir, User, Password, Host, Port, ReserveNumFiles)
		time.Sleep(5*time.Second)
	}
}