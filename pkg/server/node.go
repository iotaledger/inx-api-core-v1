package server

//nolint:unparam // even if the error is never used, the structure of all routes should be the same
func (s *DatabaseServer) info() (*infoResponse, error) {
	syncState := s.Database.LatestSyncState()

	return &infoResponse{
		Name:                        s.AppInfo.Name,
		Version:                     s.AppInfo.Version,
		IsHealthy:                   true,
		NetworkID:                   s.NetworkIDName,
		Bech32HRP:                   string(s.Bech32HRP),
		MinPoWScore:                 0,
		MessagesPerSecond:           0,
		ReferencedMessagesPerSecond: 0,
		ReferencedRate:              100,
		LatestMilestoneTimestamp:    syncState.LatestMilestoneTimestamp,
		LatestMilestoneIndex:        syncState.LatestMilestoneIndex,
		ConfirmedMilestoneIndex:     syncState.ConfirmedMilestoneIndex,
		PruningIndex:                syncState.PruningIndex,
		Features:                    []string{},
	}, nil
}
