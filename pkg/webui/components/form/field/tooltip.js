// Copyright © 2021 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import React from 'react'
import { defineMessages } from 'react-intl'

import Tooltip from '@ttn-lw/components/tooltip'
import Icon from '@ttn-lw/components/icon'
import Link from '@ttn-lw/components/link'

import Message from '@ttn-lw/lib/components/message'

import PropTypes from '@ttn-lw/lib/prop-types'
import fieldDescriptions from '@ttn-lw/lib/field-description-messages'

import style from './field.styl'

const m = defineMessages({
  descriptionTitle: 'What is it?',
  locationTitle: 'Where can I find it?',
  absenceTitle: "What if I can't find it?",
  viewGlossaryPage: 'View glossary page',
})

const Content = props => {
  const { tooltipInfo, glossaryId } = props
  const { description, location, absence } = tooltipInfo

  const hasLocation = Boolean(location)
  const hasAbsence = Boolean(absence)

  return (
    <div>
      <Message className={style.tooltipTitle} content={m.descriptionTitle} component="h4" />
      <Message className={style.tooltipDescription} content={description} component="p" />
      {hasLocation && (
        <>
          <Message className={style.tooltipTitle} content={m.locationTitle} component="h4" />
          <Message className={style.tooltipDescription} content={location} component="p" />
        </>
      )}
      {hasAbsence && (
        <>
          <Message className={style.tooltipTitle} content={m.absenceTitle} component="h4" />
          <Message className={style.tooltipDescription} content={absence} component="p" />
        </>
      )}
      <Link.GlossaryLink term={m.viewGlossaryPage} glossaryId={glossaryId} />
    </div>
  )
}

Content.propTypes = {
  glossaryId: PropTypes.string.isRequired,
  tooltipInfo: PropTypes.shape({
    description: PropTypes.message.isRequired,
    location: PropTypes.message,
    absence: PropTypes.message,
  }).isRequired,
}

const FieldTooltip = React.memo(props => {
  const { glossaryId } = props

  const tooltipInfo = fieldDescriptions[glossaryId]
  if (!tooltipInfo) {
    return null
  }

  return (
    <Tooltip
      className={style.tooltip}
      placement="bottom-start"
      interactive
      content={<Content glossaryId={glossaryId} tooltipInfo={tooltipInfo} />}
    >
      <Icon className={style.tooltipIcon} icon="help_outline" />
    </Tooltip>
  )
})

FieldTooltip.propTypes = {
  glossaryId: PropTypes.string.isRequired,
}

export default FieldTooltip
