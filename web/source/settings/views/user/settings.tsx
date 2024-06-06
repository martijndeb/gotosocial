/*
	GoToSocial
	Copyright (C) GoToSocial Authors admin@gotosocial.org
	SPDX-License-Identifier: AGPL-3.0-or-later

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU Affero General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU Affero General Public License for more details.

	You should have received a copy of the GNU Affero General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

import React from "react";
import { useTextInput, useBoolInput } from "../../lib/form";
import useFormSubmit from "../../lib/form/submit";
import { Select, TextInput, Checkbox } from "../../components/form/inputs";
import FormWithData from "../../lib/form/form-with-data";
import Languages from "../../components/languages";
import MutationButton from "../../components/form/mutation-button";
import { useVerifyCredentialsQuery } from "../../lib/query/oauth";
import { useEmailChangeMutation, usePasswordChangeMutation, useUpdateCredentialsMutation, useUserQuery } from "../../lib/query/user";
import Loading from "../../components/loading";
import { User } from "../../lib/types/user";

export default function UserSettings() {
	return (
		<FormWithData
			dataQuery={useVerifyCredentialsQuery}
			DataForm={UserSettingsForm}
		/>
	);
}

function UserSettingsForm({ data }) {
	/* form keys
		- string source[privacy]
		- bool source[sensitive]
		- string source[language]
		- string source[status_content_type]
	 */

	const form = {
		defaultPrivacy: useTextInput("source[privacy]", { source: data, defaultValue: "unlisted" }),
		isSensitive: useBoolInput("source[sensitive]", { source: data }),
		language: useTextInput("source[language]", { source: data, valueSelector: (s) => s.source.language?.toUpperCase() ?? "EN" }),
		statusContentType: useTextInput("source[status_content_type]", { source: data, defaultValue: "text/plain" }),
	};

	const [submitForm, result] = useFormSubmit(form, useUpdateCredentialsMutation());

	return (
		<>
			<h1>Account Settings</h1>
			<form className="user-settings" onSubmit={submitForm}>
				<div className="form-section-docs">
					<h3>Post Settings</h3>
					<a
						href="https://docs.gotosocial.org/en/latest/user_guide/posts"
						target="_blank"
						className="docslink"
						rel="noreferrer"
					>
						Learn more about these settings (opens in a new tab)
					</a>
				</div>
				<Select field={form.language} label="Default post language" options={
					<Languages />
				}>
				</Select>
				<Select field={form.defaultPrivacy} label="Default post privacy" options={
					<>
						<option value="private">Private / followers-only</option>
						<option value="unlisted">Unlisted</option>
						<option value="public">Public</option>
					</>
				}>
				</Select>
				<Select field={form.statusContentType} label="Default post (and bio) format" options={
					<>
						<option value="text/plain">Plain (default)</option>
						<option value="text/markdown">Markdown</option>
					</>
				}>
				</Select>
				<Checkbox
					field={form.isSensitive}
					label="Mark my posts as sensitive by default"
				/>
				<MutationButton
					disabled={false}
					label="Save settings"
					result={result}
				/>
			</form>
			<PasswordChange />
			<EmailChange />
		</>
	);
}

function PasswordChange() {
	const form = {
		oldPassword: useTextInput("old_password"),
		newPassword: useTextInput("new_password", {
			validator(val) {
				if (val != "" && val == form.oldPassword.value) {
					return "New password same as old password";
				}
				return "";
			}
		})
	};

	const verifyNewPassword = useTextInput("verifyNewPassword", {
		validator(val) {
			if (val != "" && val != form.newPassword.value) {
				return "Passwords do not match";
			}
			return "";
		}
	});

	const [submitForm, result] = useFormSubmit(form, usePasswordChangeMutation());

	return (
		<form className="change-password" onSubmit={submitForm}>
			<div className="form-section-docs">
				<h3>Change Password</h3>
				<a
					href="https://docs.gotosocial.org/en/latest/user_guide/settings/#password-change"
					target="_blank"
					className="docslink"
					rel="noreferrer"
				>
					Learn more about this (opens in a new tab)
				</a>
			</div>
			<TextInput
				type="password"
				name="password"
				field={form.oldPassword}
				label="Current password"
				autoComplete="current-password"
			/>
			<TextInput
				type="password"
				name="newPassword"
				field={form.newPassword}
				label="New password"
				autoComplete="new-password"
			/>
			<TextInput
				type="password"
				name="confirmNewPassword"
				field={verifyNewPassword}
				label="Confirm new password"
				autoComplete="new-password"
			/>
			<MutationButton
				disabled={false}
				label="Change password"
				result={result}
			/>
		</form>
	);
}

function EmailChange() {
	// Load existing user data.
	const { data: user, isFetching, isLoading } = useUserQuery();
	if (isFetching || isLoading) {
		return <Loading />;
	}

	if (user === undefined) {
		throw "could not fetch user";
	}

	return <EmailChangeForm user={user} />;
}

function EmailChangeForm({user}: {user: User}) {
	const form = {
		currentEmail: useTextInput("current_email", {
			defaultValue: user.email,
			nosubmit: true
		}),
		newEmail: useTextInput("new_email", {
			validator: (value: string | undefined) => {
				if (!value) {
					return "";
				}

				if (value.toLowerCase() === user.email?.toLowerCase()) {
					return "cannot change to your existing address";
				}

				if (value.toLowerCase() === user.unconfirmed_email?.toLowerCase()) {
					return "you already have a pending email address change to this address";
				}

				return "";
			},
		}),
		password: useTextInput("password"),
	};
	const [submitForm, result] = useFormSubmit(form, useEmailChangeMutation());

	return (
		<form className="change-email" onSubmit={submitForm}>
			<div className="form-section-docs">
				<h3>Change Email</h3>
				<a
					href="https://docs.gotosocial.org/en/latest/user_guide/settings/#email-change"
					target="_blank"
					className="docslink"
					rel="noreferrer"
				>
					Learn more about this (opens in a new tab)
				</a>
			</div>

			{ user.unconfirmed_email && <>
				<div className="info">
					<i className="fa fa-fw fa-info-circle" aria-hidden="true"></i>
					<b>
						You currently have a pending email address
						change to the address: {user.unconfirmed_email}
						<br />
						To confirm {user.unconfirmed_email} as your new
						address for this account, please check your email inbox.
					</b>
				</div>
			</> }

			<TextInput
				type="email"
				name="current-email"
				field={form.currentEmail}
				label="Current email address"
				autoComplete="none"
				disabled={true}
			/>

			<TextInput
				type="password"
				name="password"
				field={form.password}
				label="Current password"
				autoComplete="current-password"
			/>

			<TextInput
				type="email"
				name="new-email"
				field={form.newEmail}
				label="New email address"
				autoComplete="none"
			/>
			
			<MutationButton
				disabled={!form.password || !form.newEmail || !form.newEmail.valid}
				label="Change email address"
				result={result}
			/>
		</form>
	);
}
