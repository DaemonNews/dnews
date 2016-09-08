#!/bin/sh

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWwiOiJhYXJvbkBkYWVtb24ubmV3cyIsImV4cCI6MTQ3Mzk0OTcxOSwibmJmIjoxNDczMzQ0OTE5fQ.10EtnH73-rMtZAK6lfjy5sxG7jaIten8bCvmavEiWoE"

OK=$(curl -s -H "Authorization: ${TOKEN}" http://localhost:8080/api/status/ok)
echo $OK

WTF=$(curl -s -H "Authorization: ${TOKEN}" http://localhost:8080/api/stanus/snakes)
echo $WTF

UNAUTH=$(curl -s http://localhost:8080/api/status/ok)
echo $UNAUTH
