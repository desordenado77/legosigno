if [ $# -eq 0 ]
  then
    legosigno -b
    exit
fi

OUTPUT="$(./legosigno -c $1)"
if [ $? -eq 0 ]
then
  cd $OUTPUT
else
  echo "legosigno failed. Could not folder $1"
fi
