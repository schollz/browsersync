package main

const js = `// scroll saving
var cookieName = "page_scroll";
var expdays = 365;

// An adaptation of Dorcht's cookie functions.

function setCookie(name, value, expires, path, domain, secure) {
    if (!expires) expires = new Date();
    document.cookie = name + "=" + escape(value) + 
        ((expires == null) ? "" : "; expires=" + expires.toGMTString()) +
        ((path    == null) ? "" : "; path=" + path) +
        ((domain  == null) ? "" : "; domain=" + domain) +
        ((secure  == null) ? "" : "; secure");
}

function getCookie(name) {
    var arg = name + "=";
    var alen = arg.length;
    var clen = document.cookie.length;
    var i = 0;
    while (i < clen) {
        var j = i + alen;
        if (document.cookie.substring(i, j) == arg) {
            return getCookieVal(j);
        }
        i = document.cookie.indexOf(" ", i) + 1;
        if (i == 0) break;
    }
    return null;
}

function getCookieVal(offset) {
    var endstr = document.cookie.indexOf(";", offset);
    if (endstr == -1) endstr = document.cookie.length;
    return unescape(document.cookie.substring(offset, endstr));
}

function deleteCookie(name, path, domain) {
    document.cookie = name + "=" +
        ((path   == null) ? "" : "; path=" + path) +
        ((domain == null) ? "" : "; domain=" + domain) +
        "; expires=Thu, 01-Jan-00 00:00:01 GMT";
}

function saveScroll() {
    var expdate = new Date();
    expdate.setTime(expdate.getTime() + (expdays*24*60*60*1000)); // expiry date

    var x = document.pageXOffset || document.body.scrollLeft;
    var y = document.pageYOffset || document.body.scrollTop;
    var data = x + "_" + y;
    setCookie(cookieName, data, expdate);
}

function loadScroll() {
    var inf = getCookie(cookieName);
    if (!inf) { return; }
    var ar = inf.split("_");
    if (ar.length == 2) {
        window.scrollTo(parseInt(ar[0]), parseInt(ar[1]));
    }
}

document.addEventListener("DOMContentLoaded", function() {
    loadScroll();
});

window.addEventListener("beforeunload", function (event) {
    saveScroll();
});


// websockets 
var socket;
const socketMessageListener = (event) => {
	console.log(event.data);
	var data = JSON.parse(event.data);
	if (data.message == "reload") {
		location.reload();
	}
};
const socketOpenListener = (event) => {
    console.log('Connected');
};
const socketCloseListener = (event) => {
    if (socket) {
        console.log('Disconnected.');
    }
    var url = window.origin.replace("http", "ws") + '/ws';
    socket = new WebSocket(url);
    socket.addEventListener('open', socketOpenListener);
    socket.addEventListener('message', socketMessageListener);
    socket.addEventListener('close', socketCloseListener);
};
socketCloseListener();`

const jsFile = "dpMCmkDohB.js"

const defaultHTML = `<html>
<head>
<title></title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
body {
    margin: 40px auto;
    max-width: 650px;
    line-height: 1.6;
    font-size: 18px;
    color: #444;
    padding: 0 10px;
    color: #111;
    -webkit-font-smoothing: antialiased;
    text-rendering: optimizeLegibility;
    line-height: 1.4;
    font-size: 1rem;
    font-family: -apple-system, BlinkMacSystemFont, 'avenir next', avenir, helvetica, 'helvetica neue', ubuntu, roboto, noto, 'segoe ui', arial, sans-serif;
}

h1,
h2,
h3 {
    line-height: 1.2
}

a {
    color: #00aced;
    text-decoration-skip-ink: auto;
}

a:visited {
    color: #00aced;
}


a:hover {
    text-decoration: underline
}

pre>code {
    background: transparent;
    border: 0;
    font-size: 100%;
    margin: 0;
    padding: 0;
    white-space: pre;
    word-break: normal;
}

pre code {
    background-color: transparent;
    border: 0;
    display: inline;
    line-height: inherit;
    margin: 0;
    max-width: auto;
    overflow: visible;
    padding: 0;
    word-wrap: normal;
}

pre {
    word-wrap: normal;
    background-color: #f6f8fa;
    border-radius: 3px;
    font-size: 85%;
    line-height: 1.45;
    overflow: auto;
    padding: 16px;
}

code {
    background-color: rgba(27, 31, 35, .05);
    border-radius: 3px;
    font-size: 85%;
    margin: 0;
    padding: .2em .4em;
}
</style>
</head>

<body>
XX
</body>

</html>`

