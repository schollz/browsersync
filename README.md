# browsersync

<a href="https://travis-ci.org/schollz/browsersync"><img src="https://img.shields.io/travis/schollz/browsersync.svg?style=flat-square" alt="Build Status"></a>

This tool is a Golang version of a paired down [browsersync](https://www.browsersync.io/), which features live-reloading of HTML (the only feature I need) in a simple single-binary executable. It works by prepending the `</body>` of HTML pages with some [Javascript](https://github.com/schollz/browsersync/blob/master/data/sync.js) that caches the current scroll and also listens to a websockets for file changes to trigger a browser reload.

## Install

```
go install -v github.com/schollz/browsersync@latest
```

## Usage 

### Basic usage

You can just run `browsersync` in a directory:

```bash
$ browsersync
```

And then load your browser to `localhost:8003` which will render `index.html`. Any other URL will load the respective file on the computer.

### Rendering Markdown

You can also use this to render markdown. The simplest way is to use

```
$ browersync --index README.md
```

which will define the `README.md` to be the index page and automatically turn on Markdown rendering.

Alternatively, if you want to use your own styling, you simply make an HTML page, like `index.html` with the following:

```html
<html>
    <body>
        {{ MarkdownToHTML "../README.md" }}
    </body>
</html>
```

When you go to that page, the `browsersync` will automatically convert the specified markdown file (e.g. `README.md` in that example) and any changes in the file will be shown in the browser. See [`example`](https://github.com/schollz/browsersync/tree/master/example) for more info.


## Contributing

Pull requests are welcome. Feel free to...

- Revise documentation
- Add new features
- Fix bugs
- Suggest improvements

## License

MIT