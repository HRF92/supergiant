package core

//
// import "github.com/supergiant/supergiant/pkg/model"
//
// type PodObserver struct {
// 	core *Core
// }
//
// func (s *PodObserver) Perform() error {
// 	var kubes []*model.Kube
// 	if err := s.core.DB.Where("ready = ?", true).Preload("CloudAccount").Preload("Pods").Find(&kubes); err != nil {
// 		return err
// 	}
//
// 	for _, kube := range kubes {
//
// 		k8s := &KubernetesClient{kube}
//
// 		k8sPods, err := k8s.ListPods()
// 		if err != nil {
// 			return err
// 		}
//
// 		var namespacePodMetrics map[string][]*HeapsterMetric
// 		for _, kp := range k8sPods {
// 			if _, exists := namespaces[kp.ObjectMeta.Namespace]; !exists {
// 				namespaces[kp.ObjectMeta.Namespace] = nil
// 			}
// 		}
// 		for namespace := range namespacePodMetrics {
// 			metrics, err := k8s.ListPodHeapsterMetrics(namespace)
// 			if err != nil {
// 				return err
// 			}
// 			namespacePodMetrics[namespace] = metrics
// 		}
//
// 		for _, kpod := range k8sPods {
//
// 			// Find matching Pod record
// 			var pod *model.Pod
// 			for _, p := range kube.Pods {
// 				if kpod.ObjectMeta.Name == p.Name {
// 					pod = p
// 					break
// 				}
// 			}
//
// 			// Create record if it doesn't yet exist in DB
// 			if pod == nil {
// 				pod = &model.Pod{
// 					KubeID: kube.ID,
// 					Kube:   kube,
// 					Name:   kpod.ObjectMeta.Name,
// 				}
// 				if err := c.core.DB.Create(pod); err != nil {
// 					return err
// 				}
// 			}
//
// 			// Delete non-existent Pod from database
// 			if kpod == nil {
// 				c.core.DB.Delete(pod)
// 				continue
// 			}
//
// 			// Find matching Heapster metric
// 			var metric *HeapsterMetric
// 			for _, mtrc := range namespacePodMetrics[pod.Pod.ObjectMeta.Namespace] {
// 				if mtrc.Name == pod.Pod.ObjectMeta.Name {
// 					metric = mtrc
// 					break
// 				}
// 			}
//
// 			// Set Metrics
// 			pod.CPUUsage = metric.CPUUsage
// 			pod.RAMUsage = metric.RAMUsage
//
// 			pod.CPULimit = 0
// 			pod.RAMLimit = 0
// 			for _, container := range pod.Pod.Spec.Containers {
// 				if container.Resources != nil || container.Resources.Limits != nil {
// 					continue
// 				}
// 				pod.CPULimit += model.CoresFromString(container.Resources.Limits.CPU).Millicores
// 				pod.RAMLimit += model.BytesFromString(container.Resources.Limits.Memory).Bytes
// 			}
//
// 			// No limit set, use Node limit
// 			if pod.CPULimit == 0 || pod.RAMLimit == 0 {
// 				var node *model.Node
// 				if err := s.core.DB.Where("name = ?", pod.Spec.NodeName).First(node); err != nil {
// 					return err
// 				}
//
// 				if pod.CPULimit == 0 {
// 					pod.CPULimit = node.CPULimit
// 				}
//
// 				if pod.RAMLimit == 0 {
// 					pod.RAMLimit = node.RAMLimit
// 				}
// 			}
//
// 			if err := s.core.DB.Save(pod); err != nil {
// 				return err
// 			}
// 		}
// 	}
//
// 	return nil
// }
