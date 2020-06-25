#!/bin/bash

# This script can be executed with $./distributed_deploy.sh taskName
# available taskNames include: genAllMat, cleanAllMat


# This script has the following functions
# 1. genAllMat: Generate all needed materials for distributed deployment of raft_dlock (local)
# 2. cleanAllMat: Clean all generated materials for distributed deployment of raft_dlock (local)
# 3. scpAllMat: Secure copy all generated materials to all remote machines, no-passwd-login required (remote)
# 4. startAllService: Start the distribued lock service on all remote machines (remote)
# 5. stopAllService: Stop the distribued lock service on all remote machines (remote)
# 6. cleanAllVar: Clean Entry Database and Logs on all remote machines (remote)


# The following configs are used in generating all materials
# the p2p communication address between system nodes
p2p_address=("121.36.203.158" "121.37.166.51" "121.37.178.20" "121.36.198.5" "121.37.135.56")
# the p2p port for nodes
p2p_port="14005"
# the api-server communication address of nodes for dlock acquirers
clisrv_address=("${p2p_address[@]}")
# the api-server port of nodes for dlock acquirers
clisrv_port="24005"
# the node image name (arm64v8), you can revise it with your required image
image_name="zhengjilai/raft-based-dlock:0.0.1-arm64"
# image_name="zhengjilai/raft-based-dlock:0.0.1"


# The following configs are used only in ssh related tasks (remote deployment)
# the ssh/scp peer address, you should config no-password-login for your server before using this module
ssh_user_name=("hadoop" "hadoop" "hadoop" "hadoop" "hadoop")
# the ssh address for peers
ssh_address=("${p2p_address[@]}")
# the root dir for dlock materials
# shellcheck disable=SC2088
ssh_dlock_root_dir=("~/dlock" "~/dlock" "~/dlock" "~/dlock" "~/dlock")


function generateAllMaterials() {
    if [ -f ./template/config.template ] && [ -f ./template/docker-compose.template ] \
        && [ -f ./template/Makefile.template ] ; then

        # read template data
        raw_config_template=$(cat ./template/config.template)
        raw_docker_template=$(cat ./template/docker-compose.template)

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
              sed -e "s/%%%NODE_ID%%%/${index}/g" | \
              sed -e "s/%%%IMAGE_NAME%%%/${image_name/\//\\\/}/g" \
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
                    addr_list="${addr_list}    - \"${p2p_address[j]}:${p2p_port}\"\n"
                    addr_cli_list="${addr_cli_list}    - \"${clisrv_address[j]}:${clisrv_port}\"\n"
                fi
            done

            # begin to generate docker-compose-node*.yaml
            echo "${raw_config_template}" | \
                sed -e "s/%%%PEER_ID%%%/${id_list}/g" | \
                sed -e "s/%%%PEER_ADDRESS%%%/${addr_list}/g" | \
                sed -e "s/%%%PEER_CLI_ADDRESS%%%/${addr_cli_list}/g" | \
                sed -e "s/%%%SELF_ID%%%/${index}/g" | \
                sed -e "s/%%%SELF_ADDRESS%%%/\"0.0.0.0:${p2p_port}\"/g" | \
                sed -e "s/%%%SELF_CLI_ADDRESS%%%/\"0.0.0.0:${clisrv_port}\"/g" \
                > "node${index}/config-node.yaml"

            # copy the Makefile
            cp ./template/Makefile.template "node${index}/Makefile"

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

    # check the validity of preset parameters
    if [ ${#p2p_address[@]} -ne ${#ssh_user_name[@]} ]; then
        echo "List ssh_user_name should have the same length as p2p_address." && exit 1
    elif [ ${#p2p_address[@]} -ne ${#ssh_address[@]} ]; then
        echo "List ssh_address should have the same length as p2p_address." && exit 1
    elif [ ${#p2p_address[@]} -ne ${#ssh_dlock_root_dir[@]} ]; then
        echo "List ssh_dlock_root_dir should have the same length as p2p_address." && exit 1
    fi

    # first check all materials exist
    for i in "${!p2p_address[@]}";
    do
        index=$((i+1))
        printf "Begin to check material for node%s\n" "${index}"
        if ! [ -d "node${index}" ]; then
            printf "Material for node%s has not been generated.\n" "${index}" && exit 1
        fi
    done
    # begin to scp materials after checking
    for i in "${!p2p_address[@]}";
    do
        index=$((i+1))
        printf "Begin to scp material for node%s\n" "${index}"
        ssh "${ssh_user_name[i]}@${ssh_address[i]}" "cd ${ssh_dlock_root_dir[i]} && rm -rf nodeMat"
        scp -r "node${index}" "${ssh_user_name[i]}@${ssh_address[i]}:${ssh_dlock_root_dir[i]}"
        ssh "${ssh_user_name[i]}@${ssh_address[i]}" "cd ${ssh_dlock_root_dir[i]} && mv node${index} nodeMat"
    done

}

function startAllService() {
    # begin to start all services on remote machines
    for i in "${!p2p_address[@]}";
    do
        index=$((i+1))
        printf "Begin to start service for node%s\n" "${index}"
        ssh "${ssh_user_name[i]}@${ssh_address[i]}" "cd ${ssh_dlock_root_dir[i]}/nodeMat && make start"
    done
}

function stopAllService() {
    # begin to stop all services on remote machines
    for i in "${!p2p_address[@]}";
    do
        index=$((i+1))
        printf "Begin to stop service for node%s\n" "${index}"
        ssh "${ssh_user_name[i]}@${ssh_address[i]}" "cd ${ssh_dlock_root_dir[i]}/nodeMat && make stop"
    done
}

function cleanAllVar() {
    # begin to clean all variable files (Entry Database and Logs)
    for i in "${!p2p_address[@]}";
    do
        index=$((i+1))
        printf "Begin to clean all variable files (Entry Database and Logs) for node%s\n" "${index}"
        ssh "${ssh_user_name[i]}@${ssh_address[i]}" "cd ${ssh_dlock_root_dir[i]}/nodeMat && make clean"
    done
}

# address length check
if [ ${#p2p_address[@]} -ne ${#clisrv_address[@]} ]; then
    echo "List clisrv_address should have the same length as p2p_address." && exit 1
fi

case ${1} in
    "genAllMat")
        generateAllMaterials
        ;;
    "cleanAllMat")
        cleanAllMaterials
        ;;
    "scpAllMat")
        scpAllMaterials
        ;;
    "startAllService")
        startAllService
        ;;
    "stopAllService")
        stopAllService
        ;;
    "cleanAllVar")
        cleanAllVar
        ;;
    *)
        echo -e "No existing task called ${1}, please use the following tasks: \n"`
        `"genAllMat, cleanAllMat, scpAllMat, startAllService, stopAllService, cleanAllVar."
        ;;
esac