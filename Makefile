# Makefile — build/deploy BOP (bot-orchestration-platform + bop-dashboard) แบบ local
#
# มี 2 เส้นทาง deploy แยกอิสระจากกัน เลือกใช้ตามที่กำลังทดสอบ:
#   1. Docker Compose ล้วนๆ (`make compose-up`) — container ธรรมดา ไม่ผ่าน K8s เลย เหมาะ
#      ตอนอยากเช็คว่า image ทำงานร่วมกันได้จริงเร็วๆ ก่อนยุ่งกับ cluster
#   2. kind (`make deploy-local`) — จำลอง production K8s จริงบนเครื่อง dev (ดู
#      KUBERNETES-DEPLOYMENT-TODO.md หัวข้อ 9) ใช้ตอนอยากทดสอบ manifest/RBAC/probe จริงๆ
#
# ค่า default ของฝั่ง kind (REGISTRY ว่าง, IMAGE_TAG=dev) ตรงกับ image tag ที่
# k8s/deployments.yaml hardcode ไว้อยู่แล้ว จึงไม่ต้อง `kubectl set image` ตามหลัง build ปกติ
#
# นี่ไม่ใช่ตัวแทน Jenkins pipeline (JENKINS-CICD-TODO.md) — ไฟล์นี้มีไว้สำหรับ iterate เร็วๆ
# บนเครื่อง dev เองเท่านั้น production deploy จริงยังต้องผ่าน Jenkinsfile (Kaniko build +
# Trivy scan + approval gate) เสมอ ไม่ผ่าน Makefile นี้

SHELL := /bin/bash

# ปรับได้ผ่าน `make deploy-local IMAGE_TAG=abc123` หรือ `make push REGISTRY=myregistry.example.com`
KIND_CLUSTER  ?= bop
NAMESPACE     ?= bop-system
IMAGE_TAG     ?= dev
REGISTRY      ?=

BOP_DIR       := projects/bot-orchestration-platform
DASHBOARD_DIR := projects/bop-dashboard

# เติม REGISTRY prefix ให้ image name เฉพาะตอนตั้งค่า REGISTRY ไว้ (ใช้ตอน push จริงเท่านั้น
# — โหมด local ผ่าน kind ไม่ต้องมี registry เลย เพราะ `kind load docker-image` โหลดเข้า
# cluster ตรงๆ ได้โดยไม่ผ่าน registry)
image = $(if $(REGISTRY),$(REGISTRY)/$(1):$(IMAGE_TAG),$(1):$(IMAGE_TAG))

.PHONY: help build build-api build-worker build-dashboard \
        kind-cluster kind-load kind-delete \
        deploy deploy-local set-image rollout-status \
        logs-api logs-worker logs-dashboard \
        push clean \
        compose-up compose-down compose-logs compose-ps \
        net-create \
        jenkins-up jenkins-down jenkins-logs jenkins-password \
        gitlab-up gitlab-down gitlab-logs gitlab-password

.DEFAULT_GOAL := help

help: ## แสดงคำสั่งทั้งหมดพร้อมคำอธิบาย
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## ---- Build ----

build: build-api build-worker build-dashboard ## build image ทั้ง 3 ตัว (bop-api, bop-worker, bop-dashboard)

build-api: ## build image bop-api จาก cmd/bop/Dockerfile
	docker build -t $(call image,bop-api) -f $(BOP_DIR)/cmd/bop/Dockerfile $(BOP_DIR)

build-worker: ## build image bop-worker จาก cmd/bop-worker/Dockerfile
	docker build -t $(call image,bop-worker) -f $(BOP_DIR)/cmd/bop-worker/Dockerfile $(BOP_DIR)

build-dashboard: ## build image bop-dashboard จาก Dockerfile ของ Next.js (ต้องมี output: "standalone" ใน next.config.ts อยู่แล้ว)
	docker build -t $(call image,bop-dashboard) -f $(DASHBOARD_DIR)/Dockerfile $(DASHBOARD_DIR)

## ---- Docker Compose (container ล้วนๆ ไม่ผ่าน K8s เลย — ดู docker-compose.yml ที่ root) ----

compose-up: ## build (ถ้าโค้ดเปลี่ยน) แล้วรัน stack ทั้งหมด (temporal + bop-api + bop-worker + bop-dashboard) เป็น container เบื้องหลัง
	docker compose up --build -d

compose-down: ## หยุดและลบ container ทั้งหมดที่ compose-up สร้างไว้ (ไม่ลบ volume ของ Temporal postgres)
	docker compose down

compose-logs: ## tail log ของทุก container ในทีเดียว (Ctrl+C เพื่อออก ไม่หยุด container)
	docker compose logs -f

compose-ps: ## ดูสถานะ container ทั้งหมดของ stack นี้
	docker compose ps

## ---- Shared network (Jenkins ↔ GitLab คุยกันเองผ่าน container name ไม่ต้องพึ่ง ngrok) ----

net-create: ## สร้าง Docker network กลาง "bop-dev-net" ถ้ายังไม่มี (jenkins-up/gitlab-up เรียกให้อัตโนมัติอยู่แล้ว ไม่ต้องรันเองปกติ)
	docker network inspect bop-dev-net >/dev/null 2>&1 || docker network create bop-dev-net

## ---- Jenkins (controller เดี่ยวผ่าน Docker — ดู jenkins/docker-compose.yml) ----

jenkins-up: net-create ## รัน Jenkins controller เบื้องหลัง เข้าดูที่ http://localhost:8090 (รอ ~30s ให้บูตเสร็จก่อนเปิด)
	docker compose -f jenkins/docker-compose.yml up -d

jenkins-down: ## หยุด Jenkins (ไม่ลบ volume jenkins_home — config/job/plugin ที่ตั้งไว้ยังอยู่ครบ)
	docker compose -f jenkins/docker-compose.yml down

jenkins-logs: ## tail log ของ Jenkins (เช็คว่าบูตเสร็จหรือยัง หรือ debug ตอน plugin ติดตั้งไม่ผ่าน)
	docker compose -f jenkins/docker-compose.yml logs -f

jenkins-password: ## แสดง initial admin password สำหรับหน้า setup wizard ครั้งแรก (ใช้ได้แค่ตอนยังไม่ตั้ง admin user เองเท่านั้น)
	docker compose -f jenkins/docker-compose.yml exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword

## ---- GitLab (self-hosted CE ผ่าน Docker — ดู gitlab/docker-compose.yml) ----

gitlab-up: net-create ## รัน GitLab CE เบื้องหลัง เข้าดูที่ http://localhost:8929 (รอ 2-5 นาทีให้บูตเสร็จก่อนเปิด — หนักกว่า Jenkins มาก)
	docker compose -f gitlab/docker-compose.yml up -d

gitlab-down: ## หยุด GitLab (ไม่ลบ volume — project/user/config ที่ตั้งไว้ยังอยู่ครบ)
	docker compose -f gitlab/docker-compose.yml down

gitlab-logs: ## tail log ของ GitLab (เช็คว่าบูตเสร็จหรือยัง — ปกติจะเห็น log เยอะมากตอนบูตครั้งแรก)
	docker compose -f gitlab/docker-compose.yml logs -f

gitlab-password: ## แสดง initial root password (username เสมอคือ "root") — ใช้ได้แค่ 24 ชม.แรกหลัง reconfigure ครั้งแรกเท่านั้น
	docker compose -f gitlab/docker-compose.yml exec gitlab grep 'Password:' /etc/gitlab/initial_root_password

## ---- kind (local K8s cluster) ----

kind-cluster: ## สร้าง kind cluster ชื่อ $(KIND_CLUSTER) ถ้ายังไม่มี (ข้ามเฉยๆ ถ้ามีอยู่แล้ว กัน error ตอนรันซ้ำ)
	kind get clusters | grep -qx '$(KIND_CLUSTER)' || kind create cluster --name $(KIND_CLUSTER)

kind-load: ## โหลด image ทั้ง 3 ตัวที่ build ไว้เข้า kind cluster ตรงๆ (ไม่ต้อง push ขึ้น registry จริง)
	kind load docker-image $(call image,bop-api) --name $(KIND_CLUSTER)
	kind load docker-image $(call image,bop-worker) --name $(KIND_CLUSTER)
	kind load docker-image $(call image,bop-dashboard) --name $(KIND_CLUSTER)

kind-delete: ## ลบ kind cluster ทิ้งทั้งลูก (ล้าง state ทั้งหมด รวม PersistentVolume ที่ผูกกับ node)
	kind delete cluster --name $(KIND_CLUSTER)

## ---- Deploy (kubectl apply เข้า cluster ปัจจุบันตาม kubeconfig context — เช็คก่อนด้วย `kubectl config current-context`) ----

deploy: ## apply k8s manifest ทั้งหมดตามลำดับที่ต้อง (namespace ก่อนเสมอ) แล้วรอ rollout ให้เสร็จ
	kubectl apply -f k8s/namespaces.yaml
	kubectl apply -f k8s/rbac.yaml
	kubectl apply -f k8s/config.yaml
	kubectl apply -f k8s/resource-quota.yaml
	kubectl apply -f k8s/deployments.yaml
	kubectl apply -f k8s/services.yaml
	kubectl apply -f k8s/ingress.yaml
	$(MAKE) rollout-status

deploy-local: kind-cluster build kind-load deploy ## one-shot: สร้าง cluster (ถ้ายังไม่มี) + build + load + deploy ในคำสั่งเดียว — จุดเริ่มต้นที่ควรใช้เวลาทดสอบครั้งแรก

set-image: ## เปลี่ยน image tag ของ Deployment ที่ deploy ค้างอยู่แล้วโดยไม่ apply manifest ทั้งก้อนใหม่ (pattern เดียวกับที่ Jenkinsfile ใช้ตอน deploy จริง — ต้องตั้ง IMAGE_TAG ต่างจาก "dev" ก่อนถึงจะมีความหมาย)
	kubectl -n $(NAMESPACE) set image deployment/bop-api bop-api=$(call image,bop-api)
	kubectl -n $(NAMESPACE) set image deployment/bop-worker bop-worker=$(call image,bop-worker)
	kubectl -n $(NAMESPACE) set image deployment/bop-dashboard bop-dashboard=$(call image,bop-dashboard)
	$(MAKE) rollout-status

rollout-status: ## รอจน Deployment ทั้ง 3 ตัว rollout เสร็จ (fail ถ้าเกิน timeout 120 วินาที)
	kubectl -n $(NAMESPACE) rollout status deployment/bop-api --timeout=120s
	kubectl -n $(NAMESPACE) rollout status deployment/bop-worker --timeout=120s
	kubectl -n $(NAMESPACE) rollout status deployment/bop-dashboard --timeout=120s

## ---- Debug ----

logs-api: ## tail log ของ bop-api แบบ real-time
	kubectl -n $(NAMESPACE) logs -f deploy/bop-api

logs-worker: ## tail log ของ bop-worker แบบ real-time
	kubectl -n $(NAMESPACE) logs -f deploy/bop-worker

logs-dashboard: ## tail log ของ bop-dashboard แบบ real-time
	kubectl -n $(NAMESPACE) logs -f deploy/bop-dashboard

## ---- Registry (สำหรับ push ขึ้น registry จริงนอก kind เท่านั้น) ----

push: build ## push image ทั้ง 3 ตัวขึ้น $(REGISTRY) (ต้องตั้ง REGISTRY ก่อนเสมอ ไม่งั้น fail ทันที กันการ push แบบไม่ตั้งใจไปที่ Docker Hub public โดยไม่ระบุ registry เอง)
ifndef REGISTRY
	$(error ต้องตั้งค่า REGISTRY ก่อน เช่น `make push REGISTRY=myregistry.example.com`)
endif
	docker push $(call image,bop-api)
	docker push $(call image,bop-worker)
	docker push $(call image,bop-dashboard)

## ---- Cleanup ----

clean: ## ลบ image ที่ build ไว้ในเครื่อง (ไม่แตะ kind cluster — ใช้ kind-delete แยกถ้าอยากลบ cluster ด้วย)
	docker rmi -f $(call image,bop-api) $(call image,bop-worker) $(call image,bop-dashboard) 2>/dev/null || true
