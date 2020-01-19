# legosigno

legosigno in esperanto means bookmark


Legosigno creates a directory of folder bookmarks for fast browsing through folders.

Two different types of bookmarks are considered: Manually bookmarked folders and visited folders.

Manually bookmarked folders are those specifically selected to be bookmarked and will apear first when showing the list of bookmarks. These are stored in json format in file ~/.legosigno/bookmarks.json.

Visited folders are scored based on the amount of visits to the folder and displayed in order of number of visits.

The way to store the visited folders list is by making use of the PROMPT_COMMAND in bash, by calling "legosigno -V" in PROMPT_COMMAND. Each visited folder name is appended at the end of the file ~/.legosigno/visited_folders. This list of files is processed, which involves giving every folder a score (which is basically the amount of times it has been visited), ordered and stored in the same json files where manually bookmarked folders are stored.

In order to be able to jump to a folder a function is created in .bashrc that will change directory (cd) to the location outputed by legosigno -c #.

To ease setting this up in your console, an install command that will write into ~/.bashrc is created.
This command will create the following functions and aliases:

- cdb: This has dual purpose, with a numerical parameter it jumps to the specified bookmark. With no parameter it will bookmark the current folder
- cdl: list the current bookmarks
- cdr: Remove bookmark
