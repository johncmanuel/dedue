dedue is a load balancer written in Go.

# Styling

- Always handle errors explicitly; never ignore them with _ unless there's a deliberate, commented reason. Prefer early returns over nested if blocks.
- Use go doc and gopls for API and doc inspection
- Group const and var variables into a single declaration block using parentheses.
- Avoid using naked returns, unless the function is very short.
- Run go fmt to format Go code

# Testing

- Run go test ./... to test the source code.

# Source control

- Write a short, concise commit message summarizing what was done. Add more details in the commit description if needed.
- Then add commits to the current branch after testing contains no errors or warnings. Announce any commits added at end of your message.