{ pkgs, ... }:

let
  projectRoot = builtins.toString ./.;
in
{
  languages.go = {
    enable = true;
    enableHardeningWorkaround = true;
  };

  packages = with pkgs; [
    gopls
    go-tools
    temporal-cli
    gotools
    zellij
  ];

  processes = {
    temporal-dev-server.exec = "temporal server start-dev";

    worker = {
      exec = "${projectRoot}/yyx worker";
      process-compose = {
        availability.restart = "always";
        replicas = 1;
        depends_on."temporal-dev-server".condition = "process_started";
      };
    };
  };

  scripts.yyx.exec = "${projectRoot}/yyx";

  scripts.dev-run.exec = ''
    set -e
    echo "Building..."
    go build -o yyx .
    echo "Launching TUI..."
    exec ./yyx
  '';

  tasks = {
    "yyx:build" = {
      description = "Build the yyx binary (fast, no lint/test)";
      exec = ''
        go build -o yyx .
      '';
    };



    "yyx:test" = {
      description = "Run all tests";
      exec = ''
        go test ./...
      '';
    };

    "yyx:lint" = {
      description = "Run go vet";
      exec = ''
        go vet ./...
      '';
    };

    "yyx:check" = {
      description = "Run lint and tests";
      after = [ "yyx:lint" "yyx:test" ];
      exec = ''
        echo "All checks passed."
      '';
    };

    "yyx:install" = {
      description = "Full pipeline: lint, test, build, install";
      after = [ "yyx:check" "yyx:build" ];
      exec = ''
        install -d "$GOPATH/bin"
        install -m 755 yyx "$GOPATH/bin/yyx"
        echo "Installed yyx to $GOPATH/bin/yyx"
      '';
    };

    "utils:clean" = {
      description = "Remove build artifacts";
      exec = ''
        rm -f yyx
        go clean
      '';
    };
  };

  enterShell = ''
    echo "✦ yaketyyak dev environment"
    echo "  Go: $(go version 2>/dev/null || echo 'not found')"
    echo "  yyx: $(which yyx 2>/dev/null || echo 'run \`devenv tasks run project:setup\` first')"
    echo ""
    echo "  Available tasks:"
    echo "    devenv tasks list                          # discover all tasks"
    echo "    dev-run                                    # build and run TUI (fast)"
    echo "    devenv tasks run yyx:build                 # build only"
    echo "    devenv tasks run yyx:install               # full pipeline (lint+test+build+install)"
    echo "    devenv tasks run yyx:test                  # run tests"
    echo "    devenv tasks run yyx:lint                  # go vet"
    echo "    devenv tasks run utils:clean               # clean artifacts"
    echo ""
    echo "  Dev server:"
    echo "    devenv up                            Start dev server + worker"
    echo "    zellij --layout ${projectRoot}/devenv/zellij/layout.kdl  TUI with shell + devenv up"
    echo ""
    echo "  Workflow:"
    echo "    yyx start --repo user/repo --repo-root /path          Start a workflow"
    echo ""
    echo "  MCP server (for AI assistants):"
    echo "    devenv mcp                            stdio mode"
    echo "    devenv mcp --http 8080                HTTP mode"
  '';
}
