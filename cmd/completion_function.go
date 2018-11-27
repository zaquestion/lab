package cmd

var bashCompletionFunction = `
function __lab_completion_project {
  projects=($(lab project list $1))
  [ -z "$projects" ] || _values 'project' $projects
}

function __lab_completion_remote_branches {
  # TODO enable spaces in description
  branches=($(git branch -r -v | grep -v 'origin/HEAD' | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/  origin\/([^ ]*)[ ]*(.*)/\1:\2/' -e 's/ /_/g'))
  [ -z "$branches" ] || _describe 'branch' branches
}

function __lab_completion_issue {
  # TODO enable spaces in description
  issues=($(lab issue list | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/' -e 's/ /_/g'))
  [ -z "$issues" ] || _describe 'issue' issues
}

function __lab_completion_merge_request {
  # TODO enable spaces in description
  merge_requests=($(lab mr list | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/' -e 's/ /_/g'))
  [ -z "$merge_requests" ] || _describe 'merge request' merge_requests
}

function __lab_completion_snippet {
  # TODO enable spaces in description
  snippets=($(lab snippet list | sed -r -e 's/:/\\:/g' -e "s/'//g" -e 's/#([0-9]*) (.*)/\1:\2/' -e 's/ /_/g'))
  [ -z "$snippets" ] || _describe 'snippets' snippets
}
`
