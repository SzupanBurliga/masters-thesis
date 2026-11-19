#!/usr/bin/env bash
# teardown.sh — usuwa WSZYSTKIE zasoby środowiska pomiarowego.
# 1) terraform destroy (normalna ścieżka); 2) kontrola i awaryjne usunięcie po tagu Project przez AWS CLI
#    (gdyby stan Terraform zaginął). Wyniki skopiuj z generatora PRZED uruchomieniem (np. rsync results/).
set -uo pipefail
cd "$(dirname "$0")"
PROJECT=${PROJECT:-thesis-bench}
REGION=${REGION:-eu-central-1}

read -r -p "Usunąć wszystkie zasoby '$PROJECT' w $REGION? Wyniki skopiowane? Wpisz 'usuń': " ok
[ "$ok" = "usuń" ] || { echo "przerwano"; exit 1; }

if [ -f terraform.tfstate ]; then
  export TF_VAR_my_ip_cidr=${TF_VAR_my_ip_cidr:-0.0.0.0/32} TF_VAR_owner_email=${TF_VAR_owner_email:-none@example.com}
  terraform destroy -auto-approve || echo "UWAGA: terraform destroy nie powiódł się, przechodzę do sprzątania po tagu"
fi

command -v aws >/dev/null || { echo "brak AWS CLI: sprawdź ręcznie w konsoli EC2 (tag Project=$PROJECT)"; exit 0; }
ids=$(aws ec2 describe-instances --region "$REGION" --filters "Name=tag:Project,Values=$PROJECT" \
      "Name=instance-state-name,Values=pending,running,stopping,stopped" --query 'Reservations[].Instances[].InstanceId' --output text)
if [ -n "$ids" ]; then
  echo "Kończę instancje: $ids"
  aws ec2 terminate-instances --region "$REGION" --instance-ids $ids >/dev/null
  aws ec2 wait instance-terminated --region "$REGION" --instance-ids $ids
fi
vols=$(aws ec2 describe-volumes --region "$REGION" --filters "Name=tag:Project,Values=$PROJECT" "Name=status,Values=available" --query 'Volumes[].VolumeId' --output text)
for v in $vols; do aws ec2 delete-volume --region "$REGION" --volume-id "$v"; done
for sg in $(aws ec2 describe-security-groups --region "$REGION" --filters "Name=group-name,Values=$PROJECT-sg" --query 'SecurityGroups[].GroupId' --output text); do
  aws ec2 delete-security-group --region "$REGION" --group-id "$sg" || true
done
aws ec2 delete-placement-group --region "$REGION" --group-name "$PROJECT-cluster" 2>/dev/null || true
aws ec2 delete-key-pair --region "$REGION" --key-name "$PROJECT-key" 2>/dev/null || true
echo "Pozostałe zasoby z tagiem Project=$PROJECT:"
aws resourcegroupstaggingapi get-resources --region "$REGION" --tag-filters "Key=Project,Values=$PROJECT" --query 'ResourceTagMappingList[].ResourceARN' --output text
echo "Budżet (${PROJECT}-monthly) jest bezpłatny i zostaje, chyba że usunął go terraform destroy."
