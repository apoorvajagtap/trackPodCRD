#!/bin/bash
PARENT_DIR="$1"
OBJ="$2"
LOWER_OBJECT=$(echo ${OBJ} | awk '{print tolower($0)}')

Help()
{
    echo "Please pass the path to cloned repository & objects to be created as an argument."
    echo -e "\nhack/setup.sh arg1 arg2, where;"
    echo -e "arg1 = path to cloned repo (pass '.' if pwd == cloned_repo)."
    echo -e "arg2 = any of the options ('all' or 'pcrd' or 'tcrd' or 'pcr')"
    echo -e "\nFor example; hack/setup_pipelineTask.sh . all"
    exit 1
}


if [[ ${PARENT_DIR} == "" || ${OBJ} == "" ]]
then
    Help
fi

if [[ ${LOWER_OBJECT} = "pcrd" || ${LOWER_OBJECT} = "all" ]]
then
    echo -e "\n>> Creating the PipelineRun CRD"
    kubectl apply -f ${PARENT_DIR}/manifests/pipelineRun_crd.yaml
    if [ $? != 0 ]
    then
        Help
        exit 1
    fi
    echo -e "\n===================================================="
fi

if [[ ${LOWER_OBJECT} = "tcrd" || ${LOWER_OBJECT} = "all" ]]
then
    echo -e "\n>> Creating the TaskRun CRD"
    kubectl apply -f ${PARENT_DIR}/manifests/taskRun_crd.yaml
    if [ $? != 0 ]
    then
        Help
        exit 1
    fi
    echo -e "\n===================================================="    
fi

echo -e "[*] Checking the CRD details:"
kubectl api-resources | grep -i 'pipelinerun\|taskrun'
if [ $? != 0 ]
then
    echo -e "\nPlease check if the respective CRD is already present or not. If not, pass 'pcrd' or 'tcrd' or 'all' as the second argument to script."
    Help
    exit 1
fi
echo -e "\n===================================================="

if [[ ${LOWER_OBJECT} = "pcr" || ${LOWER_OBJECT} = "all" ]]
then
    echo -e "[*] Creating the PipelineRun CR in current namespace"
    kubectl apply -f ${PARENT_DIR}/manifests/prun.yaml
    echo -e "\n===================================================="
fi