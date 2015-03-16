IFS=" ";
cat hack/pull-requests | while read -ra line
do
	ID=${line[0]#*#}
	REPO=${line[1]}

	PR=$(wget -qO- https://api.github.com/repos/$REPO/pulls/$ID)
	PATCH_URL=$(echo $PR | jq -r .patch_url)
	echo $PATCH_URL

	# patch
	wget -qO- "$PATCH_URL" | patch -p 1

	# echo "github.com/$REPO/pull/$ID.patch"
	# LABEL=$(wget -qO- https://api.github.com/repos/$REPO/pulls/$ID | jq -r .head.label)
	# LABEL=${LABEL%\"}
	# LABEL=${LABEL:1}
	# echo $LABEL
done
