# Setting up the software

## Easy option

I have configured this code so it can easily run on Replit.
Just click the button below, and follow the instructions.
Remember to follow the rest of the Organizer Handbook afterwards. \
[![Run on Repl.it](https://replit.com/badge/github/sushovande/qr-mixer-game)](https://replit.com/new/github/sushovande/qr-mixer-game)



## Option 2: Hosting on your own server

QR Mixer Game can be hosted on your own website. The easiest way to do this is to host it on some cloud provider like Google Cloud or Amazon AWS. Once you're logged in to your server, do the following:

1. [Install Go](https://go.dev/doc/install)
2. Prepare to install SQLite. The go version of the sqlite driver is a CGO enabled package, so it needs the following steps:
    - `set CGO_ENABLED=1`
    - Make sure gcc is on your path. On Linux you don't need to do anything. On Windows, where having gcc in the path is uncommon, you may have to install the 64-bit gcc from a source like [tdm-gcc](https://jmeubank.github.io/tdm-gcc/).
3. Clone this repo
4. Run `go build`
    - This should automatically fetch all go dependencies, including sqlite, and compile the code
    - Optionally, you can also run `go test ./...` to check if everything is working fine.
5. Run `qr-mixer-game –port=80`
6. Verify that the site is working by visiting http://localhost.
7. This game must be served using https, because of its use of the camera API. An easy way to do that is to 
  [set up cloudflare for your site](https://developers.cloudflare.com/fundamentals/get-started/setup/add-site/) 
  and [force HTTPS](https://developers.cloudflare.com/ssl/edge-certificates/encrypt-visitor-traffic/). 
  Alternatively, you can use [Let's Encrypt](https://letsencrypt.org/getting-started/), or any other method to set up SSL.

For reliability, you should [set up this game as a service](https://medium.com/@benmorel/creating-a-linux-service-with-systemd-611b5c8b91d6).
If you use this same server for other websites also, then
[set up a reverse proxy to it](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/). 

Note that this site has no authentication. There are no passwords whatsoever, so make sure you're not expecting any security or authentication here, for any of the data.

## Navigation
 * Next page: [Setting up your questions](setting-questions.md)