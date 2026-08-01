# Volume 8 — Kubernetes

ระบบ orchestration ที่เป็นมาตรฐานอุตสาหกรรมสำหรับ deploy และจัดการ container ในระดับ production เป็นหัวใจของ Final Project ในหลักสูตรนี้

## เป้าหมายการเรียนรู้

- อธิบายและใช้งาน core object หลักของ K8s ได้ (Pod, Deployment, Service, Ingress)
- จัดการ configuration/secret แยกจากโค้ดตามหลัก 12-factor app
- เลือกใช้ workload type ให้เหมาะกับลักษณะแอป (stateless vs stateful)
- เข้าใจ storage layer และ autoscaling
- อธิบายการทำงานภายในของ control plane ได้ (etcd, scheduler) เพียงพอจะ debug ปัญหาระดับคลัสเตอร์

## สารบัญ

### [บทที่ 1 — Pod, Deployment, Service, Ingress](01-workloads-and-networking.md)

- [ ] **Pod** — หน่วยที่เล็กที่สุดใน K8s, หนึ่ง pod มีได้หลาย container (sidecar pattern)
- [ ] **Deployment** — ReplicaSet, rolling update, rollback
- [ ] **Service** — ClusterIP, NodePort, LoadBalancer, service discovery ผ่าน DNS ภายในคลัสเตอร์
- [ ] **Ingress** — routing เข้าคลัสเตอร์จากภายนอก, Ingress Controller (เชื่อมกับ [Volume 10 — APISIX](../10-apisix/README.md) ที่ใช้เป็น ingress ได้)

### [บทที่ 2 — ConfigMap, Secret](02-configuration.md)

- [ ] แยก configuration ออกจาก image (env var, mounted file)
- [ ] Secret — เก็บข้อมูล sensitive, การเข้ารหัส at-rest, ข้อจำกัด (base64 ไม่ใช่การเข้ารหัส) เชื่อมกับ [Volume 15 — Security](../15-security/README.md) (Vault)

### [บทที่ 3 — StatefulSet, DaemonSet, PVC, StorageClass](03-stateful-workloads-and-storage.md)

- [ ] **StatefulSet** — stable network identity, ordered deployment/scaling, ใช้กับ stateful workload (database)
- [ ] **DaemonSet** — รัน 1 pod ต่อ node เสมอ ใช้กับ log collector, monitoring agent
- [ ] PersistentVolume (PV) vs PersistentVolumeClaim (PVC)
- [ ] StorageClass, dynamic provisioning
- [ ] Access mode: ReadWriteOnce, ReadOnlyMany, ReadWriteMany

### [บทที่ 4 — HPA, etcd, Scheduler](04-scaling-and-control-plane.md)

- [ ] Horizontal Pod Autoscaler — scale ตาม CPU/memory หรือ custom metric
- [ ] ความสัมพันธ์กับ resource request/limit
- [ ] **etcd** — distributed key-value store ที่เก็บ state ทั้งคลัสเตอร์, consistency model
- [ ] **Scheduler** — เลือก node ให้ pod อย่างไร (filtering + scoring), affinity/anti-affinity, taint & toleration

## Lab ที่เกี่ยวข้อง

- [labs/kubernetes/](../../labs/kubernetes/README.md)

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
