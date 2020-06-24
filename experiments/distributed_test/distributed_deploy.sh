#!/bin/bash

# This script can be executed with $./distributed_deploy.sh taskName
# available taskNames include: genAllMat, cleanAllMat

# This script has the following functions
# 1. genAllMat: Generate all needed materials for distributed deployment of raft_dlock
# 2. cleanAllMat: Clean all generated materials for distributed deployment of raft_dlock

# the p2p communication address between system nodes
p2p_address=("192.168.0.1" "192.168.0.2" "192.168.0.3")
# the p2p port for nodes
p2p_port="14005"
# the client-server communication address of nodes for dlock acquirers
clisrv_address=("202.130.58.1" "202.130.58.2" "202.130.58.3")
# the client-server port of nodes for dlock acquirers
clisrv_port="24005"

# the ssh/scp peer address, you should config no-password-login for your server before using this module
ssh_address=()
# private key file, used for ssh no-password-login
private_key_file=""

function generateAllMaterials() {
    if [ -f ./template/config-template.yaml ] && [ -f ./template/docker-compose-template.yaml ] \
        && [ -f ./template/Makefile-template ] ; then

        # read template data
        raw_config_template=$(cat ./template/config-template.yaml)
        raw_docker_template=$(cat ./template/docker-compose-template.yaml)

        for i in "${!p2p_address[@]}";
        do
            index=$((i+1))
            printf "Begin to generate material for node%s\n" "${index}"
            # test if obsolete materials exists, if yes delete them
            if [ -d "node${index}" ]; then
                echo "Obsolete materials still exists, please first clean them with task cleanAllMat if you do not need them."
                exit 1
            fi
            # mkdir for new materials, at least var dir is required
            mkdir -p "node${index}/var"

            # begin to generate docker-compose-node.yaml
            echo "${raw_docker_template}" | \
              sed -e "s/%%%DOCKER_COMPOSE_PORTS%%%/\"${clisrv_port}:${clisrv_port}\"\n      - \"${p2p_port}:${p2p_port}\"\n/g" | \
              sed -e "s/%%%NODE_ID%%%/${index}/g" \
               > "node${index}/docker-compose-node.yaml"

            # begin to generate config-node.yaml
            # the following variables are used for lists in config file
            id_list="\n"
            addr_list="\n"
            addr_cli_list="\n"
            for j in "${!p2p_address[@]}";
            do
                index_inner=$((j+1))
                if [ ${index} -ne ${index_inner} ]; then
                    id_list="${id_list}    - ${index_inner}\n"
                    addr_list="${addr_list}    - \"${p2p_address[j]}\"\n"
                    addr_cli_list="${addr_cli_list}    - \"${clisrv_address[j]}\"\n"
                fi
            done

            # begin to generate docker-compose-node*.yaml
            echo "${raw_config_template}" | \
                sed -e "s/%%%PEER_ID%%%/${id_list}/g" | \
                sed -e "s/%%%PEER_ADDRESS%%%/${addr_list}/g" | \
                sed -e "s/%%%PEER_CLI_ADDRESS%%%/${addr_cli_list}/g" | \
                sed -e "s/%%%SELF_ID%%%/${index}/g" | \
                sed -e "s/%%%SELF_ADDRESS%%%/${p2p_address[i]}/g" | \
                sed -e "s/%%%SELF_CLI_ADDRESS%%%/${clisrv_address[i]}/g" \
                > "node${index}/config-node.yaml"

            # copy the Makefile
            cp ./template/Makefile-template "node${index}/Makefile"

        done

    else
        echo "Template files do not exist in ./template, thus generateAllMaterials stops."
        exit
    fi
}

function cleanAllMaterials() {
    for i in "${!p2p_address[@]}";
    do
        index=$((i+1))
        printf "Begin to clean material for node%s\n" "${index}"
        # test if obsolete materials exists, if yes delete them
        if [ -d "node${index}" ]; then
            rm -rf "node${index}"
            printf "Clean obsolete material for node%s succeeded\n" "${index}"
        fi

    done
}

function scpAllMaterials() {
    echo "hh"
}


test ${#p2p_address[@]} -ne ${#clisrv_address[@]} && \
    echo "List p2p_address should have the same length as clisrv_address." && exit 1

case ${1} in
    "genAllMat")
        generateAllMaterials
        ;;
    "cleanAllMat")
        cleanAllMaterials
        ;;
    "scpAllMat")
        cleanAllMaterials
        ;;
    *)
        echo "No existing task called $1, please use genAllMat, cleanAllMat, scpAllMat."
        ;;
esac