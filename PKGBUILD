pkgname=gonx
pkgver=0.0.1
pkgrel=1
pkgdesc='Simple reverse proxy server'
arch=('x86_64' 'aarch64')
url="https://github.com/nxshock/$pkgname"
license=('MIT')
makedepends=('go' 'git')
source=("$url/$pkgname-$pkgver.tar.gz")
sha256sums=('SKIP')
backup=("etc/$pkgname.conf")

build() {
	cd "$pkgname-$pkgver"
	go build -o "$pkgname" -ldflags "-linkmode=external -s -w" -buildmode=pie -trimpath  -mod=readonly -modcacherw
}

package() {
	cd "$pkgname-$pkgver"
	install -Dm755 "$pkgname" "$pkgdir"/usr/bin/$pkgname
	install -Dm644 "$pkgname.conf" "$pkgdir/etc/$pkgname.conf"
	install -Dm755 $pkgname.service "$pkgdir"/usr/lib/systemd/system/$pkgname.service
}
