<div align="center">

<h1><b>TermDict</b></h1>

<p>
<b>Use dictionary in the terminal</b>
</p>

</div>


![ScreenShot](https://github.com/Yodeman/termdict/assets/59335237/01b8da72-58ce-48de-8dea-45cf169dee74)

## Installation
TermDict is reasonable stable for personal use, and is being used while being developed.

[Available for download in releases](https://github.com/yodeman/termdict/releases)

#### Suggestions for better file organization and usage
TermDict uses the following path to store app data:

    - Linux and Mac -> `$HOME/.local/termdict`
    - Windows -> `%userprofile%\AppData\Local\termdict`

The executable can be placed in a child directory `bin` in the recommended app directory. 

You can then add the `bin` directory path to the `$PATH` environmental variable. Allowing you to
run the application from any terminal session.

[Here is a simple guide on how to append to $PATH](https://gist.github.com/nex3/c395b2f8fd4b02068be37c961301caa7)

## Build

- Mininum supported `go` version: `1.21.0`
  - See [Install Go](https://go.dev/doc/install)

## Credits
The following projects have been (and are still) crucial to the developement of this projec:

- [tview](https://github.com/rivo/tview)
- [TOPTED](https://www.mso.anu.edu.au/~ralph/OPTED/)
