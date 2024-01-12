package topology

import (
	"fmt"

	"github.com/flanksource/canary-checker/pkg"
	"github.com/flanksource/commons/collections/set"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func matchComponentsInRelationships(components pkg.Components, relationships []models.ComponentRelationship) error {
	if len(components) != len(relationships) {
		return fmt.Errorf("length of components & relationships should be equal")
	}

	cset := set.New[string]()
	for _, c := range components {
		cset.Add(c.ID.String())
	}
	for _, r := range relationships {
		cset.Remove(r.ComponentID.String())
	}
	if len(cset) != 0 {
		return fmt.Errorf("mismatch in ids: %s", cset)
	}
	return nil
}

var _ = ginkgo.FDescribe("Topology relationships", ginkgo.Ordered, func() {
	agent := models.Agent{ID: uuid.New(), Name: "agent"}
	topology := pkg.Topology{Name: "Topology ComponentRelationship"}

	parentComponent := pkg.Component{
		Name: "Component",
		Selectors: []types.ResourceSelector{
			{
				LabelSelector: "service=payments",
				FieldSelector: "type=api",
			},
		},
	}
	parentComponent2 := pkg.Component{
		Name: "Component2",
		Selectors: []types.ResourceSelector{
			{
				FieldSelector: "type=api,agent_id=all",
			},
		},
	}
	parentComponent3 := pkg.Component{
		Name: "Component3",
		Selectors: []types.ResourceSelector{
			{
				FieldSelector: "type=api",
			},
		},
	}
	parentComponent4 := pkg.Component{
		Name: "Component4",
		Selectors: []types.ResourceSelector{
			{
				LabelSelector: "service=logistics",
			},
		},
	}
	parentComponent5 := pkg.Component{
		Name: "Component5",
		Selectors: []types.ResourceSelector{
			{
				FieldSelector: "agent_id=" + agent.ID.String(),
			},
		},
	}
	childrenComponents := pkg.Components{
		{
			Name:   "Child-1",
			Labels: map[string]string{"service": "payments"},
			Type:   "api",
		},
		{
			Name:   "Child-2",
			Labels: map[string]string{"service": "logistics"},
		},
		{
			Name:   "Child-3",
			Labels: map[string]string{"service": "payments"},
			Type:   "api",
		},
		{
			Name:   "Child-4",
			Labels: map[string]string{"service": "payments"},
			Type:   "ui",
		},
		{
			Name:   "Child-5",
			Labels: map[string]string{"service": "logistics"},
			Type:   "api",
		},
		{
			Name:    "Child-6",
			AgentID: agent.ID,
			Labels:  map[string]string{"service": "logistics"},
			Type:    "api",
		},
	}

	ginkgo.BeforeAll(func() {
		err := DefaultContext.DB().Create(&topology).Error
		Expect(err).To(BeNil())
		err = DefaultContext.DB().Create(&agent).Error
		Expect(err).To(BeNil())

		parentComponent.TopologyID = topology.ID
		parentComponent2.TopologyID = topology.ID
		parentComponent3.TopologyID = topology.ID
		parentComponent4.TopologyID = topology.ID
		parentComponent5.TopologyID = topology.ID
		ComponentRelationshipSync.Context = DefaultContext
		err = DefaultContext.DB().Create(pkg.Components{&parentComponent, &parentComponent2, &parentComponent3, &parentComponent4, &parentComponent5}).Error
		Expect(err).To(BeNil())

		for _, c := range childrenComponents {
			c.TopologyID = topology.ID
		}
		err = DefaultContext.DB().Create(childrenComponents).Error
		Expect(err).To(BeNil())
	})

	ginkgo.It("should create component relationships", func() {
		ComponentRelationshipSync.Run()
		expectJobToPass(ComponentRelationshipSync)

		relationships, err := parentComponent.GetChildren(DefaultContext.DB())
		Expect(err).To(BeNil())

		// Child-1 and Child-3
		Expect(len(relationships)).To(Equal(2))
		Expect(matchComponentsInRelationships(pkg.Components{childrenComponents.Find("Child-1"), childrenComponents.Find("Child-3")}, relationships)).To(BeNil())

		relationships, err = parentComponent2.GetChildren(DefaultContext.DB())
		Expect(err).To(BeNil())

		// Child-1 Child-3 Child-5 Child-6
		Expect(matchComponentsInRelationships(pkg.Components{
			childrenComponents.Find("Child-1"), childrenComponents.Find("Child-3"),
			childrenComponents.Find("Child-5"), childrenComponents.Find("Child-6")}, relationships)).
			To(BeNil())
		Expect(len(relationships)).To(Equal(4))

		relationships, err = parentComponent3.GetChildren(DefaultContext.DB())
		Expect(err).To(BeNil())

		// Child-1 Child-3 Child-5
		Expect(matchComponentsInRelationships(pkg.Components{
			childrenComponents.Find("Child-1"), childrenComponents.Find("Child-3"),
			childrenComponents.Find("Child-5")}, relationships)).
			To(BeNil())
		Expect(len(relationships)).To(Equal(3))

		relationships, err = parentComponent4.GetChildren(DefaultContext.DB())
		Expect(err).To(BeNil())

		// Child-2 Child-5
		Expect(matchComponentsInRelationships(pkg.Components{childrenComponents.Find("Child-2"), childrenComponents.Find("Child-5")}, relationships)).To(BeNil())
		Expect(len(relationships)).To(Equal(2))

		relationships, err = parentComponent5.GetChildren(DefaultContext.DB())
		Expect(err).To(BeNil())

		// Child-6
		Expect(matchComponentsInRelationships(pkg.Components{childrenComponents.Find("Child-6")}, relationships)).To(BeNil())
		Expect(len(relationships)).To(Equal(1))

	})

	ginkgo.It("should handle component relationship deletions", func() {
		err := DefaultContext.DB().Delete(&childrenComponents[2]).Error
		Expect(err).To(BeNil())

		ComponentRelationshipSync.Run()
		expectJobToPass(ComponentRelationshipSync)

		relationships, err := parentComponent.GetChildren(DefaultContext.DB())
		Expect(err).To(BeNil())

		// Only child 1 should be present
		Expect(len(relationships)).To(Equal(1))

		err = DefaultContext.DB().Create(&childrenComponents[2]).Error
		Expect(err).To(BeNil())

		ComponentRelationshipSync.Run()
		expectJobToPass(ComponentRelationshipSync)

		relationships, err = parentComponent.GetChildren(DefaultContext.DB())
		Expect(err).To(BeNil())

		// Child-1 and Child-3 should be present but not Child-2
		Expect(len(relationships)).To(Equal(2))

	})

	ginkgo.It("should handle component label updates", func() {
		childrenComponents[2].Labels = map[string]string{"service": "logistics"}
		err := DefaultContext.DB().Save(&childrenComponents[2]).Error
		Expect(err).To(BeNil())
		ComponentRelationshipSync.Run()
		expectJobToPass(ComponentRelationshipSync)

		relationships, err := parentComponent.GetChildren(DefaultContext.DB())
		Expect(err).To(BeNil())

		// Only child 1 should be present as we updated the labels
		// and old relationship should be deleted
		Expect(len(relationships)).To(Equal(1))
	})
})
