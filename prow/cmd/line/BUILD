package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_binary",
    "go_library",
    "go_test",
)

go_binary(
    name = "line",
    library = ":go_default_library",
    tags = ["automanaged"],
)

go_test(
    name = "go_default_test",
    srcs = ["main_test.go"],
    library = ":go_default_library",
    tags = ["automanaged"],
    deps = [
        "//prow/github:go_default_library",
        "//prow/github/fakegithub:go_default_library",
        "//prow/jobs:go_default_library",
    ],
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    tags = ["automanaged"],
    deps = [
        "//prow/github:go_default_library",
        "//prow/jenkins:go_default_library",
        "//prow/jobs:go_default_library",
        "//prow/kube:go_default_library",
        "//prow/line:go_default_library",
        "//prow/plugins:go_default_library",
        "//vendor:github.com/Sirupsen/logrus",
    ],
)
