package grapher

import (
	"strconv"
)

// Relation describes the relationship between k8s components. Type is one of CostrolRel, LabelRel, AnnotationsRel, SpecRel.
// Source and Target contains the ID of the k8s component that is either the giver or recipient of a relationship.
// All relations are bi-directional in that each object contains both the incoming and outbound relationships.
type Relation struct {
	Source int
	Target int
}

// ControlRel describes the relationship between a controller and its children pod.
type ControlRel struct {
	Relation
	Replicas int
	Template map[string]interface{}
}

// LabelRel connects objects with spec.selector with pods that have corresponding metadata.labels.
type LabelRel struct {
	Relation
}

// ParsedObjs has methods GetControlRel and GetLabelRel that updates its objects array.
type ParsedObjs struct {
	Objects []Object
}

// Relations is embedded into the Object struct and contains arrays of the three types of relationships.
type Relations struct {
	ControlRels []ControlRel
	LabelRels   []LabelRel
}

// MatchLabel is used to match Equality label selector.
type MatchLabel struct {
	key   string
	value string
}

// MatchExpression is used to match Set-based label selectors.
type MatchExpression struct {
	key      string
	operator string // In, NotIn, Exists, DoesNotExist are valid
	values   []string
}

// =============== helpers for parsing relationships from YAML ===============

// GetControlRel generates relationships and children objects for common k8s controller types.
// Note that this only includes controllers whose children are 1) pods and 2) do not have its own YAML.
// i.e. Children relies entirely on the parent's template. Controllers like CronJob are excluded because its children are not pods.
func (parsed *ParsedObjs) GetControlRel() {
	// First collect all children (Pods) that are not included in the yaml as top-level object.
	children := []Object{}
	for i, obj := range parsed.Objects {
		yaml := obj.RawYAML

		switch kind := getField(yaml, "kind").(string); kind {
		// Parse for all possible controller types
		case "Deployment", "StatefulSet", "ReplicaSet", "DaemonSet", "Job":
			rs := getField(yaml, "spec", "replicas")

			if rs != nil && rs.(int) > 0 {
				// Add Pods for controller objects
				template := getField(yaml, "spec", "template").(map[string]interface{})
				for j := 0; j < rs.(int); j++ {
					cid := len(parsed.Objects) + len(children)
					crel := ControlRel{
						Relation: Relation{
							Source: obj.ID,
							Target: cid,
						},
						Replicas: rs.(int),
					}

					pod := Object{
						ID:      cid,
						Kind:    "Pod",
						Name:    obj.Name + "-" + strconv.Itoa(j), // tentative name pre-deploy
						RawYAML: template,
						Relations: Relations{
							ControlRels: []ControlRel{
								crel,
							},
						},
					}

					children = append(children, pod)
					obj.Relations.ControlRels = append(obj.Relations.ControlRels, crel)
					parsed.Objects[i] = obj
				}
			}
		}
	}

	// add children to the objects array at the end.
	parsed.Objects = append(parsed.Objects, children...)
}

// GetLabelRel is generates relationships between objects connected by selector-label.
// It supports both Equality-based and Set-based operators with MatchLabels and MatchExpressions, respectively.
func (parsed *ParsedObjs) GetLabelRel() {
	for i, o := range parsed.Objects {
		// Skip Pods
		yaml := o.RawYAML
		matchLabels := []MatchLabel{}
		matchExpressions := []MatchExpression{}

		// First check for the outdated syntax (matchLabels were added in recent k8s version)
		if l := getField(yaml, "spec", "selector"); l != nil {
			simple := true
			if ml := getField(yaml, "spec", "selector", "matchLabels"); ml != nil {
				matchLabels = addMatchLabels(matchLabels, ml.(map[string]interface{}))
				simple = false
			}

			if me := getField(yaml, "spec", "selector", "matchExpressions"); me != nil {
				for _, o := range me.([]interface{}) {
					ot := o.(map[string]interface{})
					values := []string{}
					for _, arg := range ot["values"].([]interface{}) {
						values = append(values, arg.(string))
					}
					matchExpressions = append(matchExpressions, MatchExpression{
						key:      ot["key"].(string),
						operator: ot["operator"].(string),
						values:   values,
					})
				}
				simple = false
			}

			if simple {
				matchLabels = addMatchLabels(matchLabels, l.(map[string]interface{}))
			}
		}

		// Find ID's of targets that match the label selector
		targetID := parsed.findLabelsBySelector(o.ID, matchLabels, matchExpressions)
		lrels := o.Relations.LabelRels
		for _, tid := range targetID {
			newrel := LabelRel{
				Relation{
					Source: o.ID,
					Target: tid,
				},
			}
			lrels = append(lrels, newrel)
		}

		parsed.Objects[i].Relations.LabelRels = lrels
	}
}

func addMatchLabels(matchLabels []MatchLabel, ml map[string]interface{}) []MatchLabel {
	for k, v := range ml {
		matchLabels = append(matchLabels, MatchLabel{
			key:   k,
			value: v.(string),
		})
	}
	return matchLabels
}

// TODO: Implement MatchExpression for set based operations.
func (parsed *ParsedObjs) findLabelsBySelector(parentID int, ml []MatchLabel, me []MatchExpression) []int {
	matchedObjs := []int{}
	for i, o := range parsed.Objects {

		// Only Pods can be selected by spec.selector
		if o.Kind != "Pod" {
			continue
		}

		// find Pods that match labels
		labels := getField(o.RawYAML, "metadata", "labels")
		match := 0
		for _, l := range ml {
			if labels.(map[string]interface{})[l.key] == l.value {
				match++
			}
		}

		// Returns only if labels meet all conditions of the selector.
		if match == len(ml) && match > 0 {
			newrel := LabelRel{
				Relation{
					Source: parentID,
					Target: o.ID,
				},
			}

			// Add bidirectional link from children as well.
			parsed.Objects[i].Relations.LabelRels = append(parsed.Objects[i].Relations.LabelRels, newrel)
			matchedObjs = append(matchedObjs, o.ID)
		}
	}
	return matchedObjs
}