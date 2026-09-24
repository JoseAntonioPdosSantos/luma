package mongodb

import "go.mongodb.org/mongo-driver/mongo/options"

// mongoReturnAfter requests the post-update document from a
// FindOneAndUpdate call, so repositories can return the persisted state
// without a second round trip.
func mongoReturnAfter() *options.FindOneAndUpdateOptions {
	return options.FindOneAndUpdate().SetReturnDocument(options.After)
}
