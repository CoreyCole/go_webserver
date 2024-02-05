# heading 1

## heading 2
1) ol 1
1) ol 2
1) ol 3

### heading 3
* ul 1
* ul 2
* ul 3

#### heading 4
- ul 4
  - ul 4.1
    - ul 4.1.1
  - ul 4.2
- ul 5
  - ul 5.1
    - ul 5.1.1
      - ul 5.1.1.1
    - ul 5.1.2
  - ul 5.2
    - ul 5.2.1
- ul 3


[this is a link]("/md/test.md")


code block in go:
```go
var n = nil
if err != nil {
    fmt.Printf("%s", err)
}
```
great code up there what about `sql`?
```sql
SELECT id, * FROM TABLE WHERE id=id
```
and a longer `python` example?
```python
from typing import List


def helper(S, i, slate, results):
    # base case
    if i == len(S):
        results.append(slate[:])  # copy slate into results
        return
    # count duplicates
    count = 0
    for j in range(i, len(S)):
        if S[i] == S[j]:
            count += 1
        else:
            break
    # exclude choice
    helper(S, i + count, slate, results)
    # include choices for all combinations of duplicates
    for _ in range(1, count + 1):
        slate.append(S[i])
        helper(S, i + count, slate, results)
    for _ in range(1, count + 1):
        slate.pop()


class Solution:
    def subsetsWithDup(self, nums: List[int]) -> List[List[int]]:
        nums.sort()
        results: List[List[int]] = []
        helper(nums, 0, [], results)
        return results
```
