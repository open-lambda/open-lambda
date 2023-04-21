// import (
//     "context"
//     "os"
//     "fmt"

//     "github.com/digitalocean/godo"
// )

// type boss_info struct {
//     ID int
//     Name string
//     Slug string
//     SSH_ID int
//     SSH_FP string
//     Memory int
//     Disk int
// }

// func main() {

//     token:=os.Getenv("DIGITALOCEAN_TOKEN")
//     client:=godo.NewFromToken(token)

//     rc, err:=init_info(client)

//     if rc != 0 {
//         fmt.Printf("ERROR: An Error has occurred!\nFunc Return Code: %v\nMsg: %v", rc, err)
//     }
//     fmt.Printf("Return Count: %v\n", rc)

// }

// func init_info(client *godo.Client) (int, error) {

//     // TODO: Cannot pass '*context.emptyCtx' type as func param
//     ctx:=context.TODO()
//     boss_idx:=0 // ASSUME: Boss is the first VM created

//     // (Droplet + SSH Key) GET req body
//     opt := &godo.ListOptions{
//         Page:    1,
//         PerPage: 200,
//     }

//     // GET: Droplet info
//     droplets, _, err := client.Droplets.List(ctx, opt)
//     if err != nil {
//         return 1, err
//     }
//     droplet:=droplets[boss_idx]

//     // GET: Key info
//     keys, _, err := client.Keys.List(ctx, opt)
//     if err != nil {
//         return 1, err
//     }
//     key:=keys[boss_idx]

//     // Fill boss struct
//     bi:=boss_info{
//         ID: droplet.ID,
//         Name: droplet.Name,
//         Slug: droplet.Region.Slug,
//         SSH_ID: key.ID,
//         SSH_FP: key.Fingerprint,
//         Memory: droplet.Memory,
//         Disk: droplet.Disk,
//     }
//     bi=bi // TODO: stubbing. Remove after click snapshot func is pushed

//     // Sanity Check: Checking information accurate
//     // fmt.Printf("ID: %v\nName: %v\nSlug: %v\nSSH_ID: %v\nSSH_FP: %v\nMemory: %v\nDisk: %v\n", bi.ID, bi.Name, bi.Slug, bi.SSH_ID, bi.SSH_FP, bi.Memory, bi.Disk)
//     // fmt.Printf("Client (type): %T\nContext (type): %T\n", client, ctx)

//     return 0, err
// }
package boss
