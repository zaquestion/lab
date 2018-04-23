#/usr/local/env sh

set -x

(cd docs && go run main.go &&
	sed -i 's|lab.md|index.md|' *.md &&
	mv lab.md index.md)
if [ ! -z ${DEPLOY} ]; then
	git config --global user.email "travis@travis-ci.org" && git config --global user.name "Travis CI"
	git remote add origin-lab https://${GITHUB_TOKEN}@github.com/zaquestion/lab.git > /dev/null 2>&1
	git fetch origin-lab && git checkout master && git add docs && git add README.md && git commit -m "(docs) ${TRAVIS_TAG}" && git push origin-lab master
fi
