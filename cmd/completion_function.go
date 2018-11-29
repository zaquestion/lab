package cmd

var zshCompletionFunction = `
function __lab_completion_remote {
  _values 'remote' $(git remote)
}

function __lab_completion_project {
  searchterm=$(echo $words[2] | sed -r 's_^-*|/$__g')
  projects=($(lab project list $searchterm))
  [ -z "$projects" ] || _values 'project' $projects
}

function __lab_completion_remote_branches {
  local remote=$@[-1]
  local IFS=$'\n'
  branches=($(git branch -r -v | grep "^ *$remote" | grep -v 'HEAD' | sed -r -e 's/:/\\:/g' -e "s/'//g" -e "s/  $remote\/([^ ]*)[ ]*(.*)/\1:\2/"))
  [ -z "$branches" ] || _describe 'branch' branches
}

function __lab_completion_issue {
  local remote=$@[-1]
  local IFS=$'\n'
  issues=($(lab issue list $remote | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/'))
  [ -z "$issues" ] || _describe 'issue' issues
}

function __lab_completion_merge_request {
  local remote=$@[-1]
  local IFS=$'\n'
  merge_requests=($(lab mr list $remote | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/'))
  [ -z "$merge_requests" ] || _describe 'merge request' merge_requests
}

function __lab_completion_snippet {
  local remote=$@[-1]
  local IFS=$'\n'
  snippets=($(lab snippet list $remote | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/'))
  [ -z "$snippets" ] || _describe 'snippet' snippets
}
`
