## lab mr show

Describe a merge request

```
lab mr show [remote] <id> [flags]
```

### Options

```
  -c, --comments        show comments for the merge request
  -h, --help            help for show
      --no-color-diff   do not show color diffs in comments
  -M, --no-markdown     don't use markdown renderer to print the issue description
  -p, --patch           show MR patches
      --reverse         reverse order when showing MR patches (chronological instead of anti-chronological)
  -s, --since string    show comments since specified date (format: 2020-08-21 14:57:46.808 +0000 UTC)
```

### Options inherited from parent commands

```
      --no-pager   Do not pipe output into a pager
```

### SEE ALSO

* [lab mr](lab_mr.md)	 - Describe, list, and create merge requests

###### Auto generated by spf13/cobra on 23-Feb-2021
