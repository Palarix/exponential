# Planned UI Improvements

## Issue Detail View
- instead of rendering in a modal dialog, we should have a dedicated single detail page
- The main content shows the issue details:
  - a header that shows the Issue title, and as a subtitle the path (Project > Parent ID (if exists) > Issue ID)
  - The description (rendered Markdown) of the issue
  - Any sub-issue if exist (as summary rows), header shows number of sub-issues and circular progress bar of sub-issues completion
  - Activity including status updates and comments (think timeline)
- On the right hand side a "permanent sidebar" shows the metadata: Status, Estimate, Labels, Dependencies