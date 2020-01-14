# legosigno

legosigno in esperanto means bookmark

File ~/.legosigno/visited_folders is used to store the folders that have been visited. You can populate the file automatically by doing: 

export PROMPT_COMMAND="echo \$PWD >> $HOME/.legosigno/visited_folders"

In order to avoid the visited_folders file to grow indefinately there is an option in legosigno to do the same but also check if the size grows too much

export PROMPT_COMMAND="legosigno -V"

legosigno processes the list, removes its content and populates the json file 