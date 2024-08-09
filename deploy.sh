#!/bin/bash


if [ "$1" == "" ]; then
echo "Usage: ./deploy.sh c  ('c' optional for check-out)"
exit
fi

if [ "$1" == "c" ]; then
echo "Check out latest version from the bitbucket"
cd /root/sakthi/go-api
git pull
fi


deploy_admin_service()
{
echo "change directory to Admin Service folder"
cd /root/sakthi/cmd/admin-service/
pwd
echo "Build Admin Service"
go build
echo "Stop Admin Service"
service admin-service stop
echo "Start Admin Service"
service admin-service start
echo "Admin Service Started"
echo "-------------------"
}

deploy_admin_service
