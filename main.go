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

	"github.com/pborman/getopt"
)


const LEGOSIGNO_ENV = "LEGOSIGNO_CONF"
const BOOKMARKS_FILENAME = "bookmarks.json"
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

func openOrCreateFile(filename string) (file *os.File, err error) {
	jsonFile, err := os.OpenFile(filename, os.O_RDWR, os.ModePerm)
	if err != nil {
		
		os.MkdirAll(strings.Replace(filename, BOOKMARKS_FILENAME, "", 1), os.ModePerm)

		Trace.Println("Unable to open bookmark file, creating it")
		jsonFile, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, os.ModePerm)
		if err != nil {
			Error.Println("Unable to create file:", err)
			return nil, err
		}
	}

	return jsonFile, nil
}

func (legosigno *Legosigno) OpenBookmarkFile() (err error) {
	var filename string

	c, exist := os.LookupEnv(LEGOSIGNO_ENV)

	if exist {
		filename = c + "/" + BOOKMARKS_FILENAME

	} else {
		filename = LEGOSIGNO_FOLDER + BOOKMARKS_FILENAME
	}
	legosigno.bookmarkFile, err = openOrCreateFile(filename)
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
	fmt.Println(legosigno.bookmarks)
	err = encoder.Encode(legosigno.bookmarks)

	if err != nil {
		fmt.Println("err: ", err)
	}
	return nil
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
	optVerbose := getopt.IntLong("Verbose", 'v', 0, "Set verbosity: 0 to 3")
	optBookmark := getopt.BoolLong("Bookmark", 'b', "Bookmark current folder")

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

	InitLogs(vt, vi, vw, os.Stderr)
	
	var legosigno Legosigno

	if *optBookmark {
		
		curr_dir, err := os.Getwd()
		if err != nil {
			Error.Println("Unable to get working directory:", err)
			os.Exit(1)
		}
		
		legosigno.OpenBookmarkFile()
		defer legosigno.bookmarkFile.Close()
		notfound := true
		for k, element := range legosigno.bookmarks.Bookmarks {
			if element.Folder == curr_dir {
				legosigno.bookmarks.Bookmarks[k].Score = legosigno.bookmarks.Bookmarks[k].Score + 1 
				notfound = false
				break
			}
		}

		if notfound {
			var e Folder
			e.Folder = curr_dir
			e.Score = 1
			legosigno.bookmarks.Bookmarks = append(legosigno.bookmarks.Bookmarks, e)
		}

		legosigno.WriteBookmarkFile()
	}

}