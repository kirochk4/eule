$os_arr = "linux", "darwin", "windows"
$arch_arr = "arm64", "amd64"
$folder = "bin"
$name = "eule"
$main = "none"

foreach ($os in $os_arr) {
    $ext = ""
    if ($os -eq "windows") {
        $ext = ".exe"
    }
    foreach ($arch in $arch_arr) {
        $env:GOOS = $os
        $env:GOARCH = $arch
        $build_name = "{0}/{1}/{2}/{3}{4}" -f $folder, $os, $arch, $name, $ext
        [System.Console]::Write("building $build_name...")
        go build -o $build_name $main
        [System.Console]::WriteLine(" done!")
    }
}
