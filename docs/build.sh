#/usr/local/env sh

(cd docs && go run main.go &&
	sed -i 's|lab.md|index.md|' *.md &&
	mv lab.md index.md)
if [ -n ${DEPLOY} ]; then
	git config --global user.email "travis@travis-ci.org" && git config --global user.name "Travis CI"
	git remote add origin-lab https://${GITHUB_TOKEN}@github.com/zaquestion/lab.git > /dev/null 2>&1
	git checkout master && git add docs && git commit -m "(docs) ${TRAVIS_TAG}" && git push origin-lab master
fi
