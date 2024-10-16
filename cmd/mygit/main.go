package main

import (
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Usage: your_program.sh <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		// Uncomment this block to pass the first stage!

		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")

	case "cat-file":
		// var arg = os.Args[2];
		var sha = os.Args[3]
		var dir_name = sha[0:2]
		var file_name = sha[2:]

		var blob_file *os.File
		var err error
		blob_file, err = os.Open("./.git/objects/" + dir_name + "/" + file_name)
		defer blob_file.Close()

		if err != nil {
			println("error while reading blob")
			os.Exit(1)
		}

		var decompressor io.ReadCloser
		decompressor, err = zlib.NewReader(blob_file)
		if err != nil {
			println("error while creating zlib reader")
		}

		var decompressed_data []byte
		decompressed_data, err = io.ReadAll(decompressor)
		if err != nil {
			println("error while reading from decompressor")
		}

		// var decompressed_text string
		var ch_rune, w = utf8.DecodeRune(decompressed_data)
		var cursor = 0
		for ch_rune != 0 {
			cursor += w
			ch_rune, w = utf8.DecodeRune(decompressed_data[cursor:])
		}
		cursor += 1

		io.WriteString(os.Stdout, string(decompressed_data[cursor:]))

	case "hash-object":
		var file_name string = os.Args[3]
		var file_content []byte
		var err error
		file_content, err = os.ReadFile(file_name)
		if err != nil {
			println("error while reading file")
			os.Exit(1)
		}

		var content_and_header string
		content_and_header = fmt.Sprintf("blob %d\x00%s", len(file_content), file_content)
		var file_content_sha = sha1.Sum([]byte(content_and_header))
		var file_content_sha_text = fmt.Sprintf("%x", file_content_sha[:])
		fmt.Println(file_content_sha_text)
		os.Mkdir("./.git/objects/"+file_content_sha_text[:2]+"/", os.ModeAppend)
		var blob_file *os.File
		blob_file, err = os.Create("./.git/objects/" + file_content_sha_text[:2] + "/" + file_content_sha_text[2:])
		if err != nil {
			fmt.Println("error while creating file")
		}

		var zlib_writer = zlib.NewWriter(blob_file)
		zlib_writer.Write([]byte("blob " + strconv.Itoa(len(file_content)) + string(0)))
		zlib_writer.Write(file_content)
		zlib_writer.Close()

	case "ls-tree":
		var object_file_name = os.Args[3]
		var obj_file, err = os.Open("./.git/objects/" + object_file_name[:2] + "/" + object_file_name[2:])
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}

		var zlib_reader io.Reader
		zlib_reader, err = zlib.NewReader(obj_file)
		if err != nil {
			println("error while creating zlib reader")
			println(err.Error())
			os.Exit(1)
		}

		var obj_file_content_decompressed []byte
		obj_file_content_decompressed, err = io.ReadAll(zlib_reader)
		if err != nil {
			println("error while reading decompressed data")
			println(err)
			os.Exit(1)
		}

		var obj_file_content_decompressed_text = string(obj_file_content_decompressed)

		for _, str := range strings.Split(obj_file_content_decompressed_text, " ")[2:] {
			fmt.Println(strings.Split(str, "\x00")[0])
		}

	case "write-tree":
		var tree_sha = write_tree(".")
		fmt.Printf("%x", string(tree_sha[:]))

	case "commit-tree":
		var tree_sha = os.Args[2]
		var parent_sha = os.Args[4]
		var message_sha = os.Args[6]
		var err error

		var obj_content = fmt.Sprintf("tree %s\nparent %s\nauthor abc abc@gmail.com\ncommitter xyz xyz@gmail.com\n\n%s\n", tree_sha, parent_sha, message_sha)
		println(obj_content)

		var obj_content_and_header = fmt.Sprintf("commit %d\x00%s", len(obj_content), obj_content)
		var obj_sha = sha1.Sum([]byte(obj_content_and_header))
		var obj_sha_text = fmt.Sprintf("%x", obj_sha)
		fmt.Println(obj_sha_text)
		var obj_path = path.Join(".", ".git", "objects", string(obj_sha_text[:2]))
		os.Mkdir(obj_path, os.ModeDir)
		obj_path = path.Join(obj_path, string(obj_sha_text[2:]))
		var obj_file *os.File 
		obj_file, err = os.Create(obj_path)
		if err != nil{
			fmt.Println(err)
			os.Exit(1)
		}
		defer obj_file.Close()

		var zlib_writer = zlib.NewWriter(obj_file)
		zlib_writer.Write([]byte(obj_content_and_header))
		zlib_writer.Close()

	case "clone":
		var url = os.Args[2]
		var dir = os.Args[3]
		os.Mkdir(dir, os.ModeDir)
		var resp, err = http.Get(url + "/info/refs?service=git-upload-pack")
		if err != nil{
			println("error when making request")
		}
		defer resp.Body.Close()
		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err != nil{
			println("error while reading body")
		}
		var main_branch_hash = strings.Split(strings.Split(string(body), "\n")[2], " ")[0][4:]

		var main_resp *http.Response
		main_resp, err = http.Post(url + "/git-upload-pack", "application/x-git-upload-pack-request", strings.NewReader("0032want " + main_branch_hash + "\n00000009done\n"))
		if err != nil{
			println("error while fetching main branch")
		}
		defer main_resp.Body.Close()
		body, err = io.ReadAll(main_resp.Body)
		if err != nil{
			println(err, "error while reading main branch response")
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}

func write_tree(path_to_dir string) [20]byte {
	var dir_content []fs.DirEntry
	var err error

	dir_content, err = os.ReadDir(path_to_dir)
	if err != nil {
		fmt.Println(err)
	}

	var tree_obj_content string = ""
	var sha string = ""
	var mode string

	for _, x := range dir_content {
		if x.Name() == ".git" {
			continue
		}

		if x.IsDir() {
			var p = path.Join(".", x.Name())
			mode = "40000"
			sha = fmt.Sprintf("%s", write_tree(p))
		} else {
			var file_content []byte
			mode = "100644"
			file_content, err = os.ReadFile(path.Join(path_to_dir, x.Name()))
			if err != nil {
				fmt.Println("error while reading file")
				fmt.Println(err)
			}

			var file_content_and_header = fmt.Sprintf("blob %d\x00%s", len(file_content), string(file_content))
			var file_content_sha = sha1.Sum([]byte(file_content_and_header))
			var file_content_sha_text = fmt.Sprintf("%x", file_content_sha[:])
			sha = string(file_content_sha[:])

			var obj_file_path = path.Join(".", ".git", "objects", file_content_sha_text[:2])
			os.Mkdir(obj_file_path, 0777)
			obj_file_path = path.Join(obj_file_path, file_content_sha_text[2:])

			var blob_file *os.File
			blob_file, err = os.Create(obj_file_path)
			if err != nil {
				fmt.Println(err)
			}
			defer blob_file.Close()

			var zlib_writer = zlib.NewWriter(blob_file)
			zlib_writer.Write([]byte(file_content_and_header))
			zlib_writer.Close()
		}
		tree_obj_content = fmt.Sprintf("%s%s %s\x00%s", tree_obj_content, mode, x.Name(), sha)
	}
	var tree_obj_sha_text string
	var tree_obj_content_and_header = fmt.Sprintf("tree %d\x00%s", len(tree_obj_content), tree_obj_content)
	var tree_obj_sha = sha1.Sum([]byte(tree_obj_content_and_header))
	tree_obj_sha_text = fmt.Sprintf("%x", tree_obj_sha[:])
	var obj_file_path string
	obj_file_path = path.Join(".", ".git", "objects", tree_obj_sha_text[:2])
	os.Mkdir(obj_file_path, 0777)
	obj_file_path = path.Join(obj_file_path, tree_obj_sha_text[2:])
	var tree_obj_file *os.File
	tree_obj_file, err = os.Create(obj_file_path)
	if err != nil {
		fmt.Println(err)
	}
	var zlib_writer = zlib.NewWriter(tree_obj_file)
	zlib_writer.Write([]byte(tree_obj_content_and_header))
	zlib_writer.Close()

	return tree_obj_sha
}