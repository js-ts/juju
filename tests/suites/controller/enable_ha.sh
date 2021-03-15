wait_for_controller_machines() {
    amount=${1}

    attempt=0
    # shellcheck disable=SC2143
    until [[ "$(juju machines -m controller --format=json | jq -r ".machines | .[] | .[\"juju-status\"] | select(.current == \"started\") | .current" | wc -l | grep "${amount}")" ]]; do
        echo "[+] (attempt ${attempt}) polling machines"
        juju machines -m controller 2>&1 | sed 's/^/    | /g'
        sleep "${SHORT_TIMEOUT}"
        attempt=$((attempt+1))

        # Wait for roughly 16 minutes for a enable-ha. In the field it's know
        # that enable-ha can take this long.
        if [[ "${attempt}" -gt 200 ]]; then
            echo "enable-ha failed waiting for machines to start"
            exit 1
        fi
    done

    if [[ "${attempt}" -gt 0 ]]; then
        echo "[+] $(green 'Completed polling machines')"
        juju machines -m controller 2>&1 | sed 's/^/    | /g'
        # Although juju reports as an idle condition, some charms require a
        # breathe period to ensure things have actually settled.
        sleep "${SHORT_TIMEOUT}"
    fi
}

run_enable_ha() {
    echo

    file="${TEST_DIR}/enable_ha.log"

    ensure "enable-ha" "${file}"

    juju deploy "cs:~jameinel/ubuntu-lite-7"

    juju enable-ha

    wait_for_controller_machines 3

    juju remove-machine -m controller 1
    juju remove-machine -m controller 2

    wait_for_controller_machines 1

    destroy_model "enable-ha"
}

test_enable_ha() {
    if [ -n "$(skip 'test_enable_ha')" ]; then
        echo "==> SKIP: Asked to skip controller enable-ha tests"
        return
    fi

    (
        set_verbosity

        cd .. || exit

        run "run_enable_ha"
    )
}
