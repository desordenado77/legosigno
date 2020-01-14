package main
import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"encoding/json"
	"os/user"
	"strings"
	"bufio"
	"strconv"

	"github.com/pborman/getopt"
)

const MAX_VISITED_FOLDERS_SIZE = 10*1024*1024 // max visited folders size 10M 
const LEGOSIGNO_ENV = "LEGOSIGNO_CONF"
const BOOKMARKS_FILENAME = "bookmarks.json"
const VISITED_FILENAME = "visited_folders"
const LEGOSIGNO_FOLDER_PATTERN = "%s/.legosigno/"
var LEGOSIGNO_FOLDER string


var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

type Folder struct {
	Folder string  `json:"folder"`
	Score int  `json:"score"`
}

type Bookmarks struct {
	Bookmarks []Folder `json:"bookmarks"`
	Visits []Folder `json:"visits"`
}

type Legosigno struct {
	bookmarks Bookmarks
	bookmarkFile *os.File
	configFolder string
	writeJson bool
}

func InitLogs(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func usage() {
	w := os.Stdout

	getopt.PrintUsage(w)
	fmt.Printf("\nExtended usage goes here\n")
}

func openOrCreateFile(filename string, mode int) (file *os.File, err error) {
	file, err = os.OpenFile(filename, mode, os.ModePerm)
	if err != nil {
		parts := strings.Split(filename, "/")
		os.MkdirAll(strings.TrimSuffix(filename, parts[len(parts)-1]), os.ModePerm)

		Trace.Println("Unable to open bookmark file, creating it")
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, os.ModePerm)
		if err != nil {
			Error.Println("Unable to create file:", err)
			return nil, err
		}
	}

	return file, nil
}


func (legosigno *Legosigno) SetConfigFolder() {
	c, exist := os.LookupEnv(LEGOSIGNO_ENV)

	if exist {
		legosigno.configFolder = c

	} else {
		legosigno.configFolder = LEGOSIGNO_FOLDER
	}
}


func (legosigno *Legosigno) OpenBookmarkFile() (err error) {
	filename := legosigno.configFolder + "/" + BOOKMARKS_FILENAME

	legosigno.bookmarkFile, err = openOrCreateFile(filename, os.O_RDWR)
	if err != nil {
		return err
	}

	byteValue, err := ioutil.ReadAll(legosigno.bookmarkFile)

	if err != nil {
		Error.Println("Unable to read bookmarks file: " + legosigno.bookmarkFile.Name())
		return err
	}

	err = json.Unmarshal(byteValue, &legosigno.bookmarks)

	if err != nil {
		Warning.Println("Unable to read json in bookmarks file: " + legosigno.bookmarkFile.Name())
	}

	return nil
}


func (legosigno *Legosigno) WriteBookmarkFile() (err error) {
	legosigno.bookmarkFile.Truncate(0)
	legosigno.bookmarkFile.Seek(0, io.SeekStart)

	encoder := json.NewEncoder(legosigno.bookmarkFile) 
	Trace.Println(legosigno.bookmarks)
	err = encoder.Encode(legosigno.bookmarks)

	if err != nil {
		Error.Println("err: ", err)
		os.Exit(1)
	}
	return nil
}

func (legosigno *Legosigno) addToVisitedFolders(folder string) (err error) {
	filename := legosigno.configFolder + "/" + VISITED_FILENAME

	visitedFile, err := openOrCreateFile(filename, os.O_RDWR | os.O_APPEND )
	if err != nil {
		return err
	}
	defer visitedFile.Close()
	
	_, err = visitedFile.WriteString(folder+"\n")

	fileInfo, err := visitedFile.Stat()
	if fileInfo.Size() >= MAX_VISITED_FOLDERS_SIZE {
		visitedFile.Close()

		legosigno.OpenBookmarkFile()
		defer legosigno.bookmarkFile.Close()

		legosigno.ProcessVisitedFolders()
		
		if legosigno.writeJson {
			legosigno.WriteBookmarkFile()
		}
	}
	
	return err
}


func (legosigno *Legosigno) ProcessVisitedFolders() (err error) {
	
	filename := legosigno.configFolder + "/" + VISITED_FILENAME

	visitedFile, err := openOrCreateFile(filename, os.O_RDWR)
	if err != nil {
		return err
	}
	defer visitedFile.Close()
 
	fileScanner := bufio.NewScanner(visitedFile)
	fileScanner.Split(bufio.ScanLines)
 
	for fileScanner.Scan() {
		folder := fileScanner.Text()
		notFound := true
		for k, element := range legosigno.bookmarks.Visits {
			if element.Folder == folder {
				legosigno.bookmarks.Visits[k].Score = legosigno.bookmarks.Visits[k].Score + 1
				legosigno.writeJson = true
				notFound = false
				break;
			}
		}
		if notFound {
			var e Folder
			e.Folder = folder
			e.Score = 1
			legosigno.bookmarks.Visits = append(legosigno.bookmarks.Visits, e)
			legosigno.writeJson = true
		}
	}

	visitedFile.Truncate(0)
	return nil
}


func removeFolder(index int, folder []Folder) []Folder {
	reader := bufio.NewReader(os.Stdin)	
	for {
		fmt.Printf("Are you sure you want to remove \"%s\" from bookmarks? (y/n)\n", folder[index].Folder)
		text, _ := reader.ReadString('\n')
		text = strings.Replace(strings.ToLower(text), "\n", "", -1)
		if text == "yes" || text == "y" {
			folder = append(folder[:index], folder[index+1:]...)
			break
		} else if text == "no" || text == "n" {
			break
		}
	}
	return folder
}

func main() {
	
	InitLogs(os.Stdout, os.Stdout, os.Stdout, os.Stderr)

	usr, err := user.Current()
	if err != nil {
		Error.Println("Error getting current user info: ")
		Error.Println(err)
		os.Exit(1)
	}

	LEGOSIGNO_FOLDER = fmt.Sprintf(LEGOSIGNO_FOLDER_PATTERN, usr.HomeDir)

	getopt.SetUsage(usage)

	optHelp := getopt.BoolLong("Help", 'h', "Show this message")
	optVisited := getopt.BoolLong("Visited", 'V', "Add current folder to visited and make sure the visited file does not grow too much")
	optVerbose := getopt.IntLong("Verbose", 'v', 0, "Set verbosity: 0 to 3. Verbose set to -1 everything goes to stderr. This is used for the cd case in which the output of the application goes to cd.")
	optBookmark := getopt.BoolLong("Bookmark", 'b', "Bookmark current folder")
	optList := getopt.BoolLong("List", 'l', "Show all bookmarks")
	optChangeDirectory := getopt.IntLong("Cd", 'c', -1, "Change to directory. This display the folder by its index. Pass the output of this command to cd to change directory like: cd \"$(./legosigno -c 0 | tail -1)\"")
	optRemoveEntry := getopt.StringLong("Remove", 'r', "-1", "Remove bookmarked folder either by index or folder name")

	getopt.Parse()

	if *optHelp {
		getopt.Usage()
		os.Exit(0)
	}

	vw := ioutil.Discard
	if *optVerbose > 0 {
		vw = os.Stdout
	}

	vi := ioutil.Discard
	if *optVerbose > 1 {
		vi = os.Stdout
	}

	vt := ioutil.Discard
	if *optVerbose > 2 {
		vt = os.Stdout
	}

	if *optVerbose == -1 {
		InitLogs(os.Stderr, os.Stderr, os.Stderr, os.Stderr)
	}

	InitLogs(vt, vi, vw, os.Stderr)
	
	var legosigno Legosigno

	legosigno.writeJson = false
	legosigno.SetConfigFolder()

	if *optVisited {
		curr_dir, err := os.Getwd()
		if err != nil {
			Error.Println("Unable to get working directory:", err)
			os.Exit(1)
		}
		legosigno.addToVisitedFolders(curr_dir)
		os.Exit(0)
	}

	legosigno.OpenBookmarkFile()
	defer legosigno.bookmarkFile.Close()

	legosigno.ProcessVisitedFolders()

	if *optBookmark {
		curr_dir, err := os.Getwd()
		if err != nil {
			Error.Println("Unable to get working directory:", err)
			os.Exit(1)
		}
		
		notfound := true
		for k, element := range legosigno.bookmarks.Bookmarks {
			if element.Folder == curr_dir {
				legosigno.bookmarks.Bookmarks[k].Score = legosigno.bookmarks.Bookmarks[k].Score + 1 
				notfound = false
				legosigno.writeJson = true
				break
			}
		}

		if notfound {
			var e Folder
			e.Folder = curr_dir
			e.Score = 1
			legosigno.bookmarks.Bookmarks = append(legosigno.bookmarks.Bookmarks, e)
			legosigno.writeJson = true
		}
	}

	if *optRemoveEntry != "-1" {
		i, err := strconv.Atoi(*optRemoveEntry)
		index := 0
		if err != nil {
			// *optRemoveEntry is a string 
			notFound := true
			for _, element := range legosigno.bookmarks.Bookmarks {
				if element.Folder == *optRemoveEntry {
					i = index
					notFound = false
					break
				}
				index = index + 1
			}
	
			if notFound {
				for _, element := range legosigno.bookmarks.Visits {
					if element.Folder == *optRemoveEntry {
						i = index
						notFound = false
						break
					}
					index = index + 1
				}
			}
			if notFound {
				Error.Printf("Folder \"%s\" Not found\n", *optRemoveEntry)
				os.Exit(1)
			}
		}
		// *optRemoveEntry is a int index 
		index = 0

		for k, _ := range legosigno.bookmarks.Bookmarks {
			if index == i {
				legosigno.bookmarks.Bookmarks = removeFolder(k, legosigno.bookmarks.Bookmarks)
				index = index + 1
				break
			}
			index = index + 1
		}
		if index <= i {
			for k, _ := range legosigno.bookmarks.Visits {
				if index == i {
					legosigno.bookmarks.Visits = removeFolder(k, legosigno.bookmarks.Visits)
					break
				}
				index = index + 1
				
			}
		}
	}

	if legosigno.writeJson {
		legosigno.WriteBookmarkFile()
	}

	if *optList {
		index := 0

		fmt.Println("Bookmarks:")
		fmt.Println("----------")
		for _, element := range legosigno.bookmarks.Bookmarks {
			fmt.Printf(" %d) %s\n", index, element.Folder)
			index = index + 1
		}
		fmt.Println()
		fmt.Println("Visited often:")
		fmt.Println("--------------")
		for _, element := range legosigno.bookmarks.Visits {
			fmt.Printf(" %d) %s  -  %d\n", index, element.Folder, element.Score)
			index = index + 1
		}
		fmt.Println()
	}

	if *optChangeDirectory != -1 {
		index := 0

		for _, element := range legosigno.bookmarks.Bookmarks {
			if index == *optChangeDirectory {
				fmt.Println(element.Folder)
				os.Exit(0)
			}
			index = index + 1
		}

		if index <= *optChangeDirectory {

			for _, element := range legosigno.bookmarks.Visits {
				if index == *optChangeDirectory {
					fmt.Println(element.Folder)
					os.Exit(0)
				}
				index = index + 1
			}
		}
		os.Exit(-1)
	}
}