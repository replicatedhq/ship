resource "local_file" "foo" {
  content     = "1"
  filename = "/tmp/foo.bar"
}
