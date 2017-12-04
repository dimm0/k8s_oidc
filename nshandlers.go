package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Process the /namespaces path
func NamespacesHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		return
	}

	session, err := filestore.Get(r, "prp-session")
	if err != nil {
		log.Printf("Error getting the session: %s", err.Error())
	}

	if session.IsNew || session.Values["userid"] == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

  // User requested to create a new namespace
  var createNsName = r.URL.Query().Get("mkns")
	if createNsName != "" {
		if _, err := clientset.Core().Namespaces().List(metav1.SingleObject(metav1.ObjectMeta{Name: createNsName})); err != nil {
			if user, err := getUser(session.Values["userid"].(string)); err == nil {
				if _, err := clientset.Core().Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: createNsName}}); err != nil {
					session.AddFlash(fmt.Sprintf("Error creating the namespace: %s", err.Error()))
					session.Save(r, w)
				} else {
          if _, err := createNsLimits(createNsName); err != nil {
            log.Printf("Error creating limits: %s", err.Error())
    			}

					_, err := createNsRoleBinding(createNsName, "admin", user)
					if err != nil {
						log.Printf("Error creating userbinding %s", err.Error())
					}
				}
			} else {
				log.Printf("Error getting the user: %s", err.Error())
			}
		} else {
			session.AddFlash(fmt.Sprintf("The namespace %s already exists", createNsName))
			session.Save(r, w)
		}
	}

  // User requested to add another user to namespace
  // var addUserName = r.URL.Query().Get("addusername"),
  //   addUserNs = r.URL.Query().Get("adduserns")
	// if addUserName != "" && addUserNs != "" {
  //
  // } else {
  //   session.AddFlash(fmt.Sprintf("Not enough arguments"))
  //   session.Save(r, w)
  // }

  namespacesList, _ := clientset.Core().Namespaces().List(metav1.ListOptions{})

	nsList := []NamespaceUserBinding{}

	for _, ns := range namespacesList.Items {
		nsBind := NamespaceUserBinding{Namespace: ns, RoleBindings: []rbacv1.RoleBinding{}}
		if nsBind.RoleBindings = getUserNamespaceBindings(session.Values["userid"].(string), ns); len(nsBind.RoleBindings) > 0 {
				nsList = append(nsList, nsBind)
		}
	}

	//Cluster ones
	nsBind := NamespaceUserBinding{ClusterRoleBindings: []rbacv1.ClusterRoleBinding{}}

  if nsBind.ClusterRoleBindings = getUserClusterBindings(session.Values["userid"].(string)); len(nsBind.ClusterRoleBindings) > 0 {
      nsList = append(nsList, nsBind)
  }

	nsVars := NamespacesTemplateVars{NamespaceBindings: nsList, IndexTemplateVars: buildIndexTemplateVars(session, w, r)}

	t, err := template.New("layout.tmpl").ParseFiles("templates/layout.tmpl", "templates/namespaces.tmpl")
	if err != nil {
		w.Write([]byte(err.Error()))
	} else {
		err = t.ExecuteTemplate(w, "layout.tmpl", nsVars)
		if err != nil {
			w.Write([]byte(err.Error()))
		}
	}
}

// Returns the rolebindings for a user and a namespace
func getUserNamespaceBindings(userId string, ns v1.Namespace) []rbacv1.RoleBinding {
	ret := []rbacv1.RoleBinding{}
	rbList, _ := clientset.Rbac().RoleBindings(ns.GetName()).List(metav1.ListOptions{})
	for _, rb := range rbList.Items {
		for _, subj := range rb.Subjects {
			var subjStr = subj.Name
			if strings.Contains(subjStr, "#") {
				subjStr = strings.Split(subjStr, "#")[1]
			}
			if subjStr == userId {
				ret = append(ret, rb)
			}
		}
	}
	return ret
}

// Returns clusterrolebindings for a user
func getUserClusterBindings(userId string) []rbacv1.ClusterRoleBinding {
	ret := []rbacv1.ClusterRoleBinding{}
  rbList, _ := clientset.Rbac().ClusterRoleBindings().List(metav1.ListOptions{})
	for _, rb := range rbList.Items {
		for _, subj := range rb.Subjects {
			var subjStr = subj.Name
			if strings.Contains(subjStr, "#") {
				subjStr = strings.Split(subjStr, "#")[1]
			}
			if subjStr == userId {
        ret = append(ret, rb)
			}
		}
	}
	return ret
}

// Creates a new rolebinding
func createNsRoleBinding(nsName string, roleName string, user PrpUser) (*rbacv1.RoleBinding, error) {
  return clientset.Rbac().RoleBindings(nsName).Create(&rbacv1.RoleBinding{
    ObjectMeta: metav1.ObjectMeta{
      Name: "cilogon",
    },
    RoleRef: rbacv1.RoleRef{
      APIGroup: "rbac.authorization.k8s.io",
      Kind:     "ClusterRole",
      Name:     roleName,
    },
    Subjects: []rbacv1.Subject{rbacv1.Subject{
      Kind: "User",
      APIGroup: "rbac.authorization.k8s.io",
      Name: user.ISS+"#"+user.UserID}},
  })
}

// Creates a namespace default limits
func createNsLimits(ns string) (*v1.LimitRange, error) {
  return clientset.Core().LimitRanges(ns).Create(&v1.LimitRange{
    ObjectMeta: metav1.ObjectMeta{Name: ns + "-mem"},
    Spec: v1.LimitRangeSpec{
      Limits: []v1.LimitRangeItem{
        v1.LimitRangeItem{
          Type: v1.LimitTypeContainer,
          Default: map[v1.ResourceName]resource.Quantity{
            v1.ResourceMemory: resource.MustParse("4Gi"),
          },
          DefaultRequest: map[v1.ResourceName]resource.Quantity{
            v1.ResourceMemory: resource.MustParse("256Mi"),
          },
        },
      },
    },
  })
}
