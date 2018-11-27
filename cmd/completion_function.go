package cmd

var zshCompletionFunction = `
function __lab_completion_project {
  projects=($(lab project list $1))
  [ -z "$projects" ] || _values 'project' $projects
}

function __lab_completion_remote_branches {
  local IFS=$'\n'
  branches=($(git branch -r -v | grep -v 'origin/HEAD' | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/  origin\/([^ ]*)[ ]*(.*)/\1:\2/'))
  [ -z "$branches" ] || _describe 'branch' branches
}

function __lab_completion_issue {
  local IFS=$'\n'
  issues=($(lab issue list | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/'))
  [ -z "$issues" ] || _describe 'issue' issues
}

function __lab_completion_merge_request {
  local IFS=$'\n'
  merge_requests=($(lab mr list | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/'))
  [ -z "$merge_requests" ] || _describe 'merge request' merge_requests
}

function __lab_completion_snippet {
  local IFS=$'\n'
  snippets=($(lab snippet list | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/'))
  [ -z "$snippets" ] || _describe 'snippets' snippets
}
`
