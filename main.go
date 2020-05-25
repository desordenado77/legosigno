package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/getopt"
)

const MAX_VISITED_FOLDERS_SIZE = 10 * 1024 * 1024 // max visited folders size 10M
const LEGOSIGNO_ENV = "LEGOSIGNO_CONF"
const BOOKMARKS_FILENAME = "bookmarks.json"
const VISITED_FILENAME = "visited_folders"
const LEGOSIGNO_FOLDER_PATTERN = "%s/.legosigno/"
const LEGOSIGNO_LOG_VISITED = "legosigno -V"
const PROMPT_COMMAND = "PROMPT_COMMAND"
const CDB_ALIAS = "cdb() { if [ $# -eq 0 ]; then legosigno -b; else OUTPUT=\"$(legosigno -c $1)\"; if [ $? -eq 0 ]; then cd $OUTPUT; else echo \"legosigno failed. Could not cd to folder $1\"; fi; fi }"
const CDL_ALIAS = "alias cdl='legosigno -l'"
const CDR_ALIAS = "alias cdr='legosigno -r'"
const LEGOSIGNO_HEADER = "################### Legosigno Start ###################"
const LEGOSIGNO_FOOTER = "###################  Legosigno End  ###################"
const MAX_AMOUNT_OF_VISITED_FOLDERS = 50
const NO_OPTION = -1000
const RESET_FONT = "\033[0m"
const BOLD_FONT = "\033[1m"
const DIM_FONT = "\033[2m"
const DARK_GREY_FONT = "\033[90m"
const LIGHT_GREY_FONT = "\033[37m"
const CYAN_FONT    = "\033[36m"
const BLUE_FONT    = "\033[34m"
const COLOR1_FONT = BLUE_FONT // DARK_GREY_FONT // BOLD_FONT
const COLOR2_FONT = CYAN_FONT // LIGHT_GREY_FONT // DIM_FONT


var LEGOSIGNO_FOLDER string

var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

type Folder struct {
	Folder string `json:"folder"`
	Score  int64  `json:"score"`
}

type Bookmarks struct {
	Bookmarks []Folder `json:"bookmarks"`
	Visits    []Folder `json:"visits"`
}

type Legosigno struct {
	bookmarks      Bookmarks
	bookmarkFile   *os.File
	configFolder   string
	writeJson      bool
	printListTo    io.Writer
	totalBookmarks int
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
	fmt.Printf("\nLegosigno creates a directory of folder bookmarks for fast browsing through folders\n")
	fmt.Printf("Two different types of bookmarks are considered: Manually bookmarked folders and visited folders\n")
	fmt.Printf("The way to store the visited folders list is by making use of the PROMPT_COMMAND in bash.\n")
	fmt.Printf("In order to be able to jump to a folder a function is created in .bashrc that will change directory (cd) to the location outputed by legosigno -c #.\n")
	fmt.Printf("To ease setting this up in your console, an install command that will write into ~/.bashrc is created.\n")
	fmt.Printf("This command will create the following functions and aliases:\n")
	fmt.Printf("\tcdb: This has dual purpose, with a numerical parameter it jumps to the specified bookmark\n")
	fmt.Printf("\t\twith no parameter it will bookmark the current folder\n")
	fmt.Printf("\tcdl: list the current bookmarks\n")
	fmt.Printf("\tcdr: Remove bookmark\n")

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

	visitedFile, err := openOrCreateFile(filename, os.O_RDWR|os.O_APPEND)
	if err != nil {
		return err
	}
	defer visitedFile.Close()

	_, err = visitedFile.WriteString(folder + " " + strconv.FormatInt(time.Now().UnixNano(), 10) + "\n")

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
		folder := strings.Split(fileScanner.Text(), " ")[0]
		time, err := strconv.ParseInt(strings.Split(fileScanner.Text(), " ")[1], 10, 64)

		if err != nil {
			Error.Panicln(err)
		}

		notFound := true
		for k, element := range legosigno.bookmarks.Visits {
			if element.Folder == folder {
				if legosigno.bookmarks.Visits[k].Score < time {
					legosigno.bookmarks.Visits[k].Score = time
					legosigno.writeJson = true
					notFound = false
					break
				}
			}
		}
		if notFound {
			var e Folder
			e.Folder = folder
			e.Score = time
			legosigno.bookmarks.Visits = append(legosigno.bookmarks.Visits, e)
			legosigno.writeJson = true
		}
	}

	visitedFile.Truncate(0)

	legosigno.bookmarks.Visits = quicksort(legosigno.bookmarks.Visits)

	if len(legosigno.bookmarks.Visits) > MAX_AMOUNT_OF_VISITED_FOLDERS {
		legosigno.bookmarks.Visits = legosigno.bookmarks.Visits[:MAX_AMOUNT_OF_VISITED_FOLDERS]
	}

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

func quicksort(a []Folder) []Folder {
	if len(a) < 2 {
		return a
	}

	left, right := 0, len(a)-1

	pivot := len(a) / 2

	a[pivot], a[right] = a[right], a[pivot]

	for i, _ := range a {
		if a[i].Score > a[right].Score {
			a[left], a[i] = a[i], a[left]
			left++
		}
	}

	a[left], a[right] = a[right], a[left]

	quicksort(a[:left])
	quicksort(a[left+1:])

	return a
}

func (legosigno *Legosigno) PrintBoookmarks() {
	legosigno.totalBookmarks = 0
	fmt.Fprintln(legosigno.printListTo)
	fmt.Fprintln(legosigno.printListTo, "Bookmarks:")
	fmt.Fprintln(legosigno.printListTo, "----------")
	for k, element := range legosigno.bookmarks.Bookmarks {
		if k%2 == 0 {
			fmt.Fprintf(legosigno.printListTo, RESET_FONT)
			fmt.Fprintf(legosigno.printListTo, COLOR1_FONT)
		} else {
			fmt.Fprintf(legosigno.printListTo, COLOR2_FONT)
		}
		fmt.Fprintf(legosigno.printListTo, " %d) %s\n", legosigno.totalBookmarks, element.Folder)
		legosigno.totalBookmarks = legosigno.totalBookmarks + 1
	}
	fmt.Fprintf(legosigno.printListTo, RESET_FONT)
	fmt.Fprintln(legosigno.printListTo)
	fmt.Fprintln(legosigno.printListTo, "Visited often:")
	fmt.Fprintln(legosigno.printListTo, "--------------")
	visitedIndex := 0
	for k, element := range legosigno.bookmarks.Visits {
		if k%2 == 0 {
			fmt.Fprintf(legosigno.printListTo, RESET_FONT)
			fmt.Fprintf(legosigno.printListTo, COLOR1_FONT)
		} else {
			fmt.Fprintf(legosigno.printListTo, COLOR2_FONT)
		}

		fmt.Fprintf(legosigno.printListTo, " %d) %s\n", legosigno.totalBookmarks, element.Folder)
		legosigno.totalBookmarks = legosigno.totalBookmarks + 1
		visitedIndex = visitedIndex + 1
		if visitedIndex >= 10 {
			break
		}
	}
	fmt.Fprintln(legosigno.printListTo)
	fmt.Fprintf(legosigno.printListTo, RESET_FONT)
	legosigno.totalBookmarks = legosigno.totalBookmarks - 1
}

func (legosigno *Legosigno) ChooseBoookmark(option string, message string) int {
	cd := NO_OPTION

	if option == "?" {
		legosigno.printListTo = os.Stderr

		legosigno.PrintBoookmarks()
		reader := bufio.NewReader(os.Stdin)
		fmt.Fprintln(legosigno.printListTo, "which folder do you want to "+message+"?\n")
		text, _ := reader.ReadString('\n')
		text = strings.Replace(strings.ToLower(text), "\n", "", -1)

		number, err := strconv.Atoi(text)
		if err == nil && number <= legosigno.totalBookmarks {
			cd = number
		} else {
			if number > legosigno.totalBookmarks {
				Error.Println("Invalid bookmark index")
			}
			if err != nil {
				Error.Println(err)
			}
			os.Exit(-1)
		}

	} else {
		if option != "" {
			number, err := strconv.Atoi(option)
			if err == nil {
				cd = number
			} else {
				Error.Println("Parameter should be number or ?")
				Error.Println(err)
				os.Exit(1)
			}
		}
	}
	return cd
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
	optChangeDirectory := getopt.StringLong("Cd", 'c', "", "Change to directory. This display the folder by its index. Pass the output of this command to cd to change directory like: cd \"$(./legosigno -c 0 | tail -1)\". Use \"?\" to show the list and type the selected folder to change to")
	optRemoveEntry := getopt.StringLong("Remove", 'r', "-1", "Remove bookmarked folder either by index or folder name. Use \"?\" to show the list and type the selected folder to change to")
	optInstall := getopt.BoolLong("Install", 'i', "Install legosigno")

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

	if *optInstall {
		// add cdb alias in bashrc to perform a cd to a bookmarked folder
		// added PROMPT_COMMAND to log visited folders

		c, exist := os.LookupEnv(PROMPT_COMMAND)

		// This is a basic-lazy check. Could be improved, but it is probably not worth it
		if !exist || (exist && -1 == strings.Index(c, LEGOSIGNO_LOG_VISITED)) {
			// Need to append to .bashrc file
			f, err := os.OpenFile(usr.HomeDir+"/.bashrc", os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				Error.Println(err)
				os.Exit(1)
			}
			defer f.Close()
			p := ""
			if exist {
				p = c + ";"
			}
			if _, err := f.WriteString("\n" + LEGOSIGNO_HEADER +
				"\nexport " + PROMPT_COMMAND + "=\"" + p + LEGOSIGNO_LOG_VISITED + "\"\n" +
				CDB_ALIAS + "\n" +
				CDR_ALIAS + "\n" +
				CDL_ALIAS + "\n" +
				LEGOSIGNO_FOOTER + "\n"); err != nil {
				log.Println(err)
			}
			fmt.Println("legosigno installed in .bashrc")
			fmt.Println("Do \"source ~/.bashrc\" to reload it")
		} else {
			fmt.Println("Nothing to do. Seems like PROMPT_COMMAND already calls legosigno")
		}
		os.Exit(0)
	}

	var legosigno Legosigno
	legosigno.printListTo = os.Stdout
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
		if err != nil && *optRemoveEntry != "?" {
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

		// *optRemoveEntry is a int index or a ?
		i = legosigno.ChooseBoookmark(*optRemoveEntry, "remove")

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

	if *optList {
		// should be able to figure out if there has been changes or not
		legosigno.writeJson = true

		legosigno.PrintBoookmarks()
	}

	if legosigno.writeJson {
		legosigno.WriteBookmarkFile()
	}

	cd := legosigno.ChooseBoookmark(*optChangeDirectory, "change to")

	if cd >= 0 {

		index := 0

		for _, element := range legosigno.bookmarks.Bookmarks {
			if index == cd {
				fmt.Println(element.Folder)
				os.Exit(0)
			}
			index = index + 1
		}

		if index <= cd {

			for _, element := range legosigno.bookmarks.Visits {
				if index == cd {
					fmt.Println(element.Folder)
					os.Exit(0)
				}
				index = index + 1
			}
		}
		os.Exit(-1)
	} else {
		if cd < 0 {
			for k, element := range legosigno.bookmarks.Visits {
				if (k + 1) == -cd {
					fmt.Println(element.Folder)
					os.Exit(0)
				}
			}
		}
	}

}
