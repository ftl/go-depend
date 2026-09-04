# Project: go-depend

## General Instructions
- You are an expert Go developer.
- DO NOT change code unrelated to the current task, even if it contains obvious flaws. When in doubt, ask the user.
- DO NOT commit anything! NEVER!
- Use the ponytail skill in full mode.
- Never assume, always investigate the locally available codebase or research online, with a very strong preference to the locally available codebase.
- If unsure about what the user wants, ask the user.
- If temporary folders or files are needed, always use `/tmp`.
- Use the [mermaid](https://github.com/mermaid-js/mermaid) graph language to visualize architectural facts and implementation details.

## Coding Style
- Write ideomatic Go code and adhere to the [Google Go Styleguide](https://google.github.io/styleguide/go/).
- Prefer direct, boring, maintainable code over hacky or magical code.
- Prefer short functions and methods. Do not mix different levels of abstraction in one function or method.
- Avoid panics!
- Use the github.com/stretchr/testify library for writing any automated tests.
- Use the github.com/spf13/cobra library for implementing the CLI part.

## Workmode
We structure the work in stories, which consist of several tasks. The smallest unit of work is one iteration. Keep iterations small and comprehensible, and contain only strongly related changes. When an iteration is finished, the code base must not produce any compiler errors.

We use the @doc/ folder to store any documents.
