git rebase --abort
git checkout master

##
# Reset repo to the upstream and update marker commit
##

# this is the commit marker commit
COMMIT=$(git log --pretty="%H" --grep="\[YONT\] add merge script")

# made the current change to keep the merge script updated
git add hack/*
git commit -q --fixup $COMMIT

echo "Rewrite history to keep the merge script updated"
EDITOR="sed -e ''" git rebase -q -i --autosquash $COMMIT^1 > /dev/null

# re-read commit ID
COMMIT=$(git log --pretty="%H" --grep="\[YONT\] add merge script")
# reset back to the marking point
git reset --hard $COMMIT > /dev/null

# rebase master to Docker Machine
git remote update origin > /dev/null

# reset back to the marking point
git reset $COMMIT^1 > /dev/null

# rewrite the new marker commit
echo "Reset repo to the upstream"
git reset --hard origin/master > /dev/null

echo "Rewrite new marker commit"
git add hack/*
git commit -q -m "[YONT] add merge script"
echo "Rewrite successfully"

##
# Start patching process
##
IFS=" ";
cat hack/pull-requests | while read -ra line
do
	# get ID and REPO from pull-requests
	if [[ ${line[0]} == \#* ]]; then
		ID=${line[0]#*#}
		REPO=${line[1]}
		git ppr $REPO $ID
	fi
done

git push yont master -f
