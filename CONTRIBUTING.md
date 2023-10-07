# Contributing to termdict

First of all, thank you for taking the time to contribute.

The following provides you with some guidance on how to contribute to this project. Mainly, it is meant to save us all some time so please read it.

Please note that this document is work in progress so I might add to it in the future.

## Issues

- Please include enough information so everybody understands your request.
- Screenshots or code that illustrates your point always helps.
- It's fine to ask for help.
- If you request a new feature, state your motivation. It should be something that others will also need.

## Pull Requests

If you have a feature request, open an issue first before sending a pull request, and allow for some discussion.

## More on contributing to dictionary words database

Each file in the [words database](https://github.com/Yodeman/termdict/tree/main/word_dbase/json) is named after the english alphabet, each containing words starting with that particular alphabet.

When contributing to the database, locate the appropriate file and the appropriate position (words are sorted in lexicogrphical order) to place the new word. The format for each word is shown below:

```json
"new word in lower case" : {
    "word" : "new word in lower case",
    "alternate_spellings": [list of alternative spellings],
    "definitions": [
        {
            "part_of_speech": "",
            "defintion": ""
        },
    ]
}
```
