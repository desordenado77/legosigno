# legosigno

legosigno in esperanto means bookmark

File ~/.legosigno/visited_folders is used to store the folders that have been visited. You can populate the file automatically by doing: 

export PROMPT_COMMAND="echo \$PWD >> $HOME/.legosigno/visited_folders"

legosigno processes the list, removes its content and populates the json file 