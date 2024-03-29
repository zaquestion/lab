## lab mr discussion

Start a discussion on an MR on GitLab

```
lab mr discussion [remote] [<MR ID or branch>] [flags]
```

### Examples

```
lab mr discussion
lab mr discussion origin
lab mr discussion my_remote -m "discussion comment"
lab mr discussion upstream -F my_comment.txt
lab mr discussion --commit abcdef123456
lab mr discussion my-topic-branch
lab mr discussion origin 123
lab mr discussion origin my-topic-branch
lab mr discussion --commit abcdef123456 --position=main.c:+100,100
```

### Options

```
  -c, --commit string     start a thread on a commit
  -F, --file string       use the given file as the message
  -h, --help              help for discussion
  -m, --message strings   use the given <msg>; multiple -m are concatenated as separate paragraphs
      --position string   start a thread on a specific line of the diff
                          argument must be of the form <file>":"["+" | "-" | " "]<old_line>","<new_line>
                          that is, the file name, followed by the line type - one of "+" (added line),
                          "-" (deleted line) or a space character (context line) - followed by
                          the line number in the old version of the file, a ",", and finally
                          the line number in the new version of the file. If the line type is "+", then
                          <old_line> is ignored. If the line type is "-", then <new_line> is ignored.
                          
                          Here's an example diff that explains how to determine the old/new line numbers:
                             
                          	--- a/README.md		old	new
                          	+++ b/README.md
                          	@@ -100,3 +100,4 @@
                          	 pre-context line	100	100
                          	-deleted line		101	101
                          	+added line 1		101	102
                          	+added line 2		101	103
                          	 post-context line	102	104
                          
                          # Comment on "deleted line":
                          lab mr discussion --commit=commit-id --position=README.md:-101,101
                          # Comment on "added line 2":
                          lab mr discussion --commit=commit-id --position=README.md:+101,103
                          # Comment on the "post-context line":
                          lab mr discussion --commit=commit-id --position=README.md:\ 102,104
```

### Options inherited from parent commands

```
      --debug      Enable debug logging level
      --no-pager   Do not pipe output into a pager
      --quiet      Turn off any sort of logging. Only command output is printed
```

### SEE ALSO

* [lab mr](lab_mr.md)	 - Describe, list, and create merge requests

###### Auto generated by spf13/cobra on 3-Jul-2022
