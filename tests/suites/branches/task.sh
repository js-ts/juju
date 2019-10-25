test_branches() {
    if [ "$(skip 'test_branches')" ]; then
        echo "==> TEST SKIPPED: branches tests"
        return
    fi

    set_verbosity

    echo "==> Checking for dependencies"
    check_dependencies juju

    file="${TEST_DIR}/test-branch.txt"

    export JUJU_DEV_FEATURE_FLAGS=generations

    bootstrap "test-branch" "${file}"

    test_branch

    destroy_controller "test-branch"
}
