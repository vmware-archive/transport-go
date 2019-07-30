package bus

import (
    "errors"
)

type ChannelManager interface {
    Boot()
    CreateChannel(channelName string) *Channel
    DestroyChannel(channelName string)
    CheckChannelExists(channelName string) bool
    GetChannel(channelName string) (*Channel, error)
    GetAllChannels() map[string]*Channel
}

type ChannelManagerImpl struct {
    Channels map[string]*Channel
}

func (manager *ChannelManagerImpl) Boot() {
    manager.Channels = make(map[string]*Channel)
}

func (manager *ChannelManagerImpl) CreateChannel(channelName string) *Channel {
    channel := &Channel{
        Name:   channelName,
        //Stream: make(chan *Message)
        }

    manager.Channels[channelName] = channel
    return channel
}

func (manager *ChannelManagerImpl) DestroyChannel(channelName string) {
    delete(manager.Channels, channelName)
}

func (manager *ChannelManagerImpl) GetChannel(channelName string) (*Channel, error) {
    if channel, ok := manager.Channels[channelName]; ok {
        return channel, nil
    } else {
        return nil, errors.New("Channel does not exist: " + channelName)
    }
}

func (manager *ChannelManagerImpl) GetAllChannels() map[string]*Channel {
    return manager.Channels
}

func (manager *ChannelManagerImpl) CheckChannelExists(channelName string) bool {
    return manager.Channels[channelName] != nil
}
